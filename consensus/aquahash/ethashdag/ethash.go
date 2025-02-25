package ethashdag

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math"
	"math/big"
	mrand "math/rand"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/edsrzf/mmap-go"
	"github.com/hashicorp/golang-lru/simplelru"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/bitutil"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/crypto/sha3"
	"gitlab.com/aquachain/aquachain/params"
)

const (
	datasetInitBytes   = 1 << 30 // Bytes in dataset at genesis
	datasetGrowthBytes = 1 << 23 // Dataset growth per epoch
	cacheInitBytes     = 1 << 24 // Bytes in cache at genesis
	cacheGrowthBytes   = 1 << 17 // Cache growth per epoch
	epochLength        = 30000   // Blocks per epoch
	mixBytes           = 128     // Width of mix
	hashBytes          = 64      // Hash length in bytes
	hashWords          = 16      // Number of 32 bit ints in a hash
	datasetParents     = 256     // Number of parents of each dataset element
	cacheRounds        = 3       // Number of rounds in cache production
	loopAccesses       = 64      // Number of accesses in hashimoto loop
)

type Lru = lru

var HashimotoFull = hashimotoFull

var ErrInvalidDumpMagic = errors.New("invalid dump magic")

var (
	// maxUint256 is a big integer representing 2^256-1
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	// algorithmRevision is the data structure version used for file naming.
	algorithmRevision = 2

	// dumpMagic is a dataset dump header to sanity check a data dump.
	dumpMagic = []uint32{0xbaddcafe, 0xfee1dead}
)

// isLittleEndian returns whether the local system is running in little or big
// endian byte order.
func isLittleEndian() bool {
	n := uint32(0x01020304)
	return *(*byte)(unsafe.Pointer(&n)) == 0x04
}

func (aquahash *EthashDAG) VerifySeal(number uint64, header *types.Header) (cache *cache, digest []byte, result []byte, err error) {
	cache = aquahash.cache(number)
	if cache == nil {
		return nil, nil, nil, errors.New("invalid startVersion for use with ethash")
	}
	size := datasetSize(number)
	if aquahash.config.PowMode == ModeTest {
		size = 32 * 1024
	}
	digest, result = hashimotoLight(size, cache.cache, header.HashNoNonce().Bytes(), header.Nonce.Uint64())
	// Caches are unmapped in a finalizer. Ensure that the cache stays live
	// until after the call to hashimotoLight so it's not unmapped while being used.
	runtime.KeepAlive(cache)
	return cache, digest, result, nil
}

// memoryMap tries to memory map a file of uint32s for read only access.
func memoryMap(path string) (*os.File, mmap.MMap, []uint32, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, 0644)
	if err != nil {
		return nil, nil, nil, err
	}
	mem, buffer, err := memoryMapFile(file, false)
	if err != nil {
		file.Close()
		return nil, nil, nil, err
	}
	for i, magic := range dumpMagic {
		if buffer[i] != magic {
			mem.Unmap()
			file.Close()
			return nil, nil, nil, ErrInvalidDumpMagic
		}
	}
	return file, mem, buffer[len(dumpMagic):], err
}

// memoryMapFile tries to memory map an already opened file descriptor.
func memoryMapFile(file *os.File, write bool) (mmap.MMap, []uint32, error) {
	// Try to memory map the file
	flag := mmap.RDONLY
	if write {
		flag = mmap.RDWR
	}
	mem, err := mmap.Map(file, flag, 0)
	if err != nil {
		return nil, nil, err
	}
	// Yay, we managed to memory map the file, here be dragons
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&mem))
	header.Len /= 4
	header.Cap /= 4

	return mem, *(*[]uint32)(unsafe.Pointer(&header)), nil
}

// memoryMapAndGenerate tries to memory map a temporary file of uint32s for write
// access, fill it with the data from a generator and then move it into the final
// path requested.
func memoryMapAndGenerate(path string, size uint64, generator func(buffer []uint32)) (*os.File, mmap.MMap, []uint32, error) {
	// Ensure the data folder exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, nil, nil, err
	}
	// Create a huge temporary empty file to fill with data
	temp := path + "." + strconv.Itoa(mrand.Intn(1000+1000))

	dump, err := os.Create(temp)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = dump.Truncate(int64(len(dumpMagic))*4 + int64(size)); err != nil {
		return nil, nil, nil, err
	}
	// Memory map the file for writing and fill it with the generator
	mem, buffer, err := memoryMapFile(dump, true)
	if err != nil {
		dump.Close()
		return nil, nil, nil, err
	}
	copy(buffer, dumpMagic)

	data := buffer[len(dumpMagic):]
	generator(data)

	if err := mem.Unmap(); err != nil {
		return nil, nil, nil, err
	}
	if err := dump.Close(); err != nil {
		return nil, nil, nil, err
	}
	if err := os.Rename(temp, path); err != nil {
		return nil, nil, nil, err
	}
	return memoryMap(path)
}

// lru tracks caches or datasets by their last use time, keeping at most N of them.
type lru struct {
	what  string
	newFn func(epoch uint64) interface{}
	mu    sync.Mutex
	// Items are kept in a LRU cache, but there is a special case:
	// We always keep an item for (highest seen epoch) + 1 as the 'future item'.
	cache      *simplelru.LRU
	future     uint64
	futureItem interface{}
}

// newlru create a new least-recently-used cache for ither the verification caches
// or the mining datasets.
func newlru(what string, maxItems int, newFn func(epoch uint64) interface{}) *lru {
	if maxItems <= 0 {
		maxItems = 1
	}
	cache, _ := simplelru.NewLRU(maxItems, func(key, value interface{}) {
		log.Trace("Evicted aquahash "+what, "epoch", key)
	})
	return &lru{what: what, newFn: newFn, cache: cache}
}

// get retrieves or creates an item for the given epoch. The first return value is always
// non-nil. The second return value is non-nil if lru thinks that an item will be useful in
// the near future.
func (lru *lru) get(epoch uint64) (item, future interface{}) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	// Get or create the item for the requested epoch.
	item, ok := lru.cache.Get(epoch)
	if !ok {
		if lru.future > 0 && lru.future == epoch {
			item = lru.futureItem
		} else {
			log.Trace("Requiring new aquahash "+lru.what, "epoch", epoch)
			item = lru.newFn(epoch)
		}
		lru.cache.Add(epoch, item)
	}
	// Update the 'future item' if epoch is larger than previously seen.
	if epoch < MaxEpoch-1 && lru.future < epoch+1 {
		log.Trace("Requiring new future aquahash "+lru.what, "epoch", epoch+1)
		future = lru.newFn(epoch + 1)
		lru.future = epoch + 1
		lru.futureItem = future
	}
	return item, future
}

// cache wraps an aquahash cache with some metadata to allow easier concurrent use.
type cache struct {
	epoch uint64    // Epoch for which this cache is relevant
	dump  *os.File  // File descriptor of the memory mapped cache
	mmap  mmap.MMap // Memory map itself to unmap before releasing
	cache []uint32  // The actual cache data content (may be memory mapped)
	once  sync.Once // Ensures the cache is generated only once
}

// newCache creates a new aquahash verification cache and returns it as a plain Go
// interface to be usable in an LRU cache.
func newCache(epoch uint64) interface{} {
	return &cache{epoch: epoch}
}

// generate ensures that the cache content is generated before use.
func (c *cache) generate(dir string, limit int, test bool) {
	c.once.Do(func() {
		size := cacheSize(c.epoch*epochLength + 1)
		seed := seedHash(c.epoch*epochLength+1, 0)
		if test {
			size = 1024
		}
		// If we don't store anything on disk, generate and return.
		if dir == "" {
			c.cache = make([]uint32, size/4)
			generateCache(c.cache, c.epoch, seed)
			return
		}
		// Disk storage is needed, this will get fancy
		var endian string
		if !isLittleEndian() {
			endian = ".be"
		}
		path := filepath.Join(dir, fmt.Sprintf("cache-R%d-%x%s", algorithmRevision, seed[:8], endian))
		logger := log.New("epoch", c.epoch)

		// We're about to mmap the file, ensure that the mapping is cleaned up when the
		// cache becomes unused.
		runtime.SetFinalizer(c, (*cache).finalizer)

		// Try to load the file from disk and memory map it
		var err error
		c.dump, c.mmap, c.cache, err = memoryMap(path)
		if err == nil {
			logger.Debug("Loaded old aquahash cache from disk")
			return
		}
		logger.Debug("Failed to load old aquahash cache", "err", err)

		// No previous cache available, create a new cache file to fill
		c.dump, c.mmap, c.cache, err = memoryMapAndGenerate(path, size, func(buffer []uint32) { generateCache(buffer, c.epoch, seed) })
		if err != nil {
			logger.Error("Failed to generate mapped aquahash cache", "err", err)

			c.cache = make([]uint32, size/4)
			generateCache(c.cache, c.epoch, seed)
		}
		// Iterate over all previous instances and delete old ones
		for ep := int(c.epoch) - limit; ep >= 0; ep-- {
			seed := seedHash(uint64(ep)*epochLength+1, 0)
			path := filepath.Join(dir, fmt.Sprintf("cache-R%d-%x%s", algorithmRevision, seed[:8], endian))
			os.Remove(path)
		}
	})
}

// finalizer unmaps the memory and closes the file.
func (c *cache) finalizer() {
	if c.mmap != nil {
		c.mmap.Unmap()
		c.dump.Close()
		c.mmap, c.dump = nil, nil
	}
}

type Dataset = dataset

// dataset wraps an aquahash dataset with some metadata to allow easier concurrent use.
type dataset struct {
	epoch   uint64    // Epoch for which this cache is relevant
	dump    *os.File  // File descriptor of the memory mapped cache
	mmap    mmap.MMap // Memory map itself to unmap before releasing
	dataset []uint32  // The actual cache data content
	once    sync.Once // Ensures the cache is generated only once
}

func (d *dataset) GetDataset() []uint32 {
	return d.dataset
}

// newDataset creates a new aquahash mining dataset and returns it as a plain Go
// interface to be usable in an LRU cache.
func newDataset(epoch uint64) interface{} {
	return &dataset{epoch: epoch}
}

// Mode defines the type and amount of PoW verification an aquahash engine makes.
type Mode uint

const (
	ModeNormal Mode = iota
	ModeShared
	ModeTest
	ModeFake
	ModeFullFake
)

// Config are the configuration parameters of the aquahash.
type Config struct {
	CacheDir       string
	CachesInMem    int
	CachesOnDisk   int
	DatasetDir     string
	DatasetsInMem  int
	DatasetsOnDisk int
	PowMode        Mode
	StartVersion   params.HeaderVersion
}

type EthashDAG struct {
	// caches is a LRU cache of verification caches.
	caches *lru
	// datasets is a LRU cache of mining datasets.
	datasets *lru
	// config is the configuration for the aquahash PoW algorithm.
	config *Config

	// lock protects the caches and datasets from concurrent access.
	lock sync.Mutex
}

func New(config *Config) *EthashDAG {
	return &EthashDAG{
		config:   config,
		datasets: newlru("dataset", config.DatasetsInMem, newDataset),
		caches:   newlru("cache", config.CachesInMem, newCache),
	}
}

// cache tries to retrieve a verification cache for the specified block number
// by first checking against a list of in-memory caches, then against caches
// stored on disk, and finally generating one if none can be found.
func (aquahash *EthashDAG) cache(block uint64) *cache {
	aquahash.lock.Lock()
	defer aquahash.lock.Unlock()
	if aquahash.caches == nil {
		aquahash.caches = newlru("cache", aquahash.config.CachesInMem, newCache)
	}
	epoch := block / epochLength
	currentI, futureI := aquahash.caches.get(epoch)
	current := currentI.(*cache)

	// Wait for generation finish.
	current.generate(aquahash.config.CacheDir, aquahash.config.CachesOnDisk, aquahash.config.PowMode == ModeTest)

	// If we need a new future cache, now's a good time to regenerate it.
	if futureI != nil {
		future := futureI.(*cache)
		go future.generate(aquahash.config.CacheDir, aquahash.config.CachesOnDisk, aquahash.config.PowMode == ModeTest)
	}
	return current
}

// dataset tries to retrieve a mining dataset for the specified block number
// by first checking against a list of in-memory datasets, then against DAGs
// stored on disk, and finally generating one if none can be found.
func (aquahash *EthashDAG) dataset(block uint64) *dataset {
	epoch := block / epochLength
	currentI, futureI := aquahash.datasets.get(epoch)
	current := currentI.(*dataset)

	// Wait for generation finish.
	current.generate(aquahash.config.DatasetDir, aquahash.config.DatasetsOnDisk, aquahash.config.PowMode == ModeTest)

	if false { // not really needed
		// If we need a new future dataset, now's a good time to regenerate it.
		if futureI != nil {
			future := futureI.(*dataset)
			go future.generate(aquahash.config.DatasetDir, aquahash.config.DatasetsOnDisk, aquahash.config.PowMode == ModeTest)
		}
	}
	return current
}

func (d *EthashDAG) Dataset(n uint64) *dataset {
	return d.dataset(n)
}

// generate ensures that the dataset content is generated before use.
func (d *dataset) generate(dir string, limit int, test bool) {
	logger := log.New("epoch", d.epoch)
	d.once.Do(func() {
		csize := cacheSize(d.epoch*epochLength + 1)
		dsize := datasetSize(d.epoch*epochLength + 1)
		seed := seedHash(d.epoch*epochLength+1, 0)
		if test {
			csize = 1024
			dsize = 32 * 1024
		}
		// If we don't store anything on disk, generate and return
		if dir == "" {
			cache := make([]uint32, csize/4)
			generateCache(cache, d.epoch, seed)

			d.dataset = make([]uint32, dsize/4)
			generateDataset(d.dataset, d.epoch, cache)
		}
		// Disk storage is needed, this will get fancy
		var endian string
		if !isLittleEndian() {
			endian = ".be"
		}
		path := filepath.Join(dir, fmt.Sprintf("full-R%d-%x%s", algorithmRevision, seed[:8], endian))

		// We're about to mmap the file, ensure that the mapping is cleaned up when the
		// cache becomes unused.
		runtime.SetFinalizer(d, (*dataset).finalizer)

		// Try to load the file from disk and memory map it
		var err error
		d.dump, d.mmap, d.dataset, err = memoryMap(path)
		if err == nil {
			logger.Debug("Loaded old aquahash dataset from disk")
			return
		}
		logger.Debug("Failed to load old aquahash dataset", "err", err)

		// No previous dataset available, create a new dataset file to fill
		cache := make([]uint32, csize/4)
		generateCache(cache, d.epoch, seed)

		d.dump, d.mmap, d.dataset, err = memoryMapAndGenerate(path, dsize, func(buffer []uint32) { generateDataset(buffer, d.epoch, cache) })
		if err != nil {
			logger.Error("Failed to generate mapped aquahash dataset", "err", err)

			d.dataset = make([]uint32, dsize/2)
			generateDataset(d.dataset, d.epoch, cache)
		}
		// Iterate over all previous instances and delete old ones
		for ep := int(d.epoch) - limit; ep >= 0; ep-- {
			seed := seedHash(uint64(ep)*epochLength+1, 0)
			path := filepath.Join(dir, fmt.Sprintf("full-R%d-%x%s", algorithmRevision, seed[:8], endian))
			os.Remove(path)
		}
	})
}

// finalizer closes any file handlers and memory maps open.
func (d *dataset) finalizer() {
	if d.mmap != nil {
		d.mmap.Unmap()
		d.dump.Close()
		d.mmap, d.dump = nil, nil
	}
}

// MakeCache generates a new aquahash cache and optionally stores it to disk.
func MakeCache(block uint64, dir string) {
	c := cache{epoch: block / epochLength}
	c.generate(dir, math.MaxInt32, false)
}

// MakeDataset generates a new aquahash dataset and optionally stores it to disk.
func MakeDataset(block uint64, dir string) {
	d := dataset{epoch: block / epochLength}
	d.generate(dir, math.MaxInt32, false)
}

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures
// to be reused between hash runs instead of requiring new ones to be created.
// The returned function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	return func(dest []byte, data []byte) {
		h.Write(data)
		h.Sum(dest[:0])
		h.Reset()
	}
}

// seedHash is the seed to use for generating a verification cache and the mining
// dataset.
func seedHash(block uint64, version byte) []byte {
	if version > 1 {
		return common.BytesToHash([]byte{version}).Bytes() // eg 0x0000....0001
	}
	seed := make([]byte, 32)
	if block < epochLength {
		return seed
	}
	keccak256 := makeHasher(sha3.NewKeccak256())
	for i := 0; i < int(block/epochLength); i++ {
		keccak256(seed, seed)
	}
	return seed
}

// generateCache creates a verification cache of a given size for an input seed.
// The cache production process involves first sequentially filling up 32 MB of
// memory, then performing two passes of Sergio Demian Lerner's RandMemoHash
// algorithm from Strict Memory Hard Hashing Functions (2014). The output is a
// set of 524288 64-byte values.
// This method places the result into dest in machine byte order.
func generateCache(dest []uint32, epoch uint64, seed []byte) {
	// Print some debug logs to allow analysis on low end devices
	logger := log.New("epoch", epoch)

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)

		logFn := logger.Debug
		if elapsed > 3*time.Second {
			logFn = logger.Info
		}
		logFn("Generated aquahash verification cache", "elapsed", common.PrettyDuration(elapsed))
	}()
	// Convert our destination slice to a byte buffer
	// Convert our destination slice to a byte buffer
	header := *(*reflect.SliceHeader)(unsafe.Pointer(&dest))
	header.Len *= 4
	header.Cap *= 4
	cache := *(*[]byte)(unsafe.Pointer(&header))

	// Calculate the number of theoretical rows (we'll store in one buffer nonetheless)
	size := uint64(len(cache))
	rows := int(size) / hashBytes

	// Start a monitoring goroutine to report progress on low end devices
	var progress uint32

	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(3 * time.Second):
				logger.Info("Generating aquahash verification cache", "percentage", atomic.LoadUint32(&progress)*100/uint32(rows)/4, "elapsed", common.PrettyDuration(time.Since(start)))
			}
		}
	}()
	// Create a hasher to reuse between invocations
	keccak512 := makeHasher(sha3.NewKeccak512())

	// Sequentially produce the initial dataset
	keccak512(cache, seed)
	for offset := uint64(hashBytes); offset < size; offset += hashBytes {
		keccak512(cache[offset:], cache[offset-hashBytes:offset])
		atomic.AddUint32(&progress, 1)
	}
	// Use a low-round version of randmemohash
	temp := make([]byte, hashBytes)

	for i := 0; i < cacheRounds; i++ {
		for j := 0; j < rows; j++ {
			var (
				srcOff = ((j - 1 + rows) % rows) * hashBytes
				dstOff = j * hashBytes
				xorOff = (binary.LittleEndian.Uint32(cache[dstOff:]) % uint32(rows)) * hashBytes
			)
			bitutil.XORBytes(temp, cache[srcOff:srcOff+hashBytes], cache[xorOff:xorOff+hashBytes])
			keccak512(cache[dstOff:], temp)

			atomic.AddUint32(&progress, 1)
		}
	}
	// Swap the byte order on big endian systems and return
	if !isLittleEndian() {
		swap(cache)
	}
}

// swap changes the byte order of the buffer assuming a uint32 representation.
func swap(buffer []byte) {
	for i := 0; i < len(buffer); i += 4 {
		binary.BigEndian.PutUint32(buffer[i:], binary.LittleEndian.Uint32(buffer[i:]))
	}
}

// fnv is an algorithm inspired by the FNV hash, which in some cases is used as
// a non-associative substitute for XOR. Note that we multiply the prime with
// the full 32-bit input, in contrast with the FNV-1 spec which multiplies the
// prime with one byte (octet) in turn.
func fnv(a, b uint32) uint32 {
	return a*0x01000193 ^ b
}

// fnvHash mixes in data into mix using the aquahash fnv method.
func fnvHash(mix []uint32, data []uint32) {
	for i := 0; i < len(mix); i++ {
		mix[i] = mix[i]*0x01000193 ^ data[i]
	}
}

// generateDatasetItem combines data from 256 pseudorandomly selected cache nodes,
// and hashes that to compute a single dataset node.
func generateDatasetItem(cache []uint32, index uint32, keccak512 hasher) []byte {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(len(cache) / hashWords)

	// Initialize the mix
	mix := make([]byte, hashBytes)

	binary.LittleEndian.PutUint32(mix, cache[(index%rows)*hashWords]^index)
	for i := 1; i < hashWords; i++ {
		binary.LittleEndian.PutUint32(mix[i*4:], cache[(index%rows)*hashWords+uint32(i)])
	}
	keccak512(mix, mix)

	// Convert the mix to uint32s to avoid constant bit shifting
	intMix := make([]uint32, hashWords)
	for i := 0; i < len(intMix); i++ {
		intMix[i] = binary.LittleEndian.Uint32(mix[i*4:])
	}
	// fnv it with a lot of random cache nodes based on index
	for i := uint32(0); i < datasetParents; i++ {
		parent := fnv(index^i, intMix[i%16]) % rows
		fnvHash(intMix, cache[parent*hashWords:])
	}
	// Flatten the uint32 mix into a binary one and return
	for i, val := range intMix {
		binary.LittleEndian.PutUint32(mix[i*4:], val)
	}
	keccak512(mix, mix)
	return mix
}

// generateDataset generates the entire aquahash dataset for mining.
// This method places the result into dest in machine byte order.
func generateDataset(dest []uint32, epoch uint64, cache []uint32) {
	// Print some debug logs to allow analysis on low end devices
	logger := log.New("epoch", epoch)

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		logFn := logger.Info
		if elapsed > 3*time.Second {
			logFn = logger.Warn
		}
		logFn("Generated aquahash verification cache", "elapsed", common.PrettyDuration(elapsed))
	}()

	// Figure out whether the bytes need to be swapped for the machine
	swapped := !isLittleEndian()

	dataset := *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(unsafe.Pointer(&dest[0])),
		Len:  len(dest) * 4,
		Cap:  len(dest) * 4,
	}))

	// Generate the dataset on many goroutines since it takes a while
	threads := runtime.NumCPU()
	size := uint64(len(dataset))

	var pend sync.WaitGroup
	pend.Add(threads)

	var progress uint32
	for i := 0; i < threads; i++ {
		go func(id int) {
			defer pend.Done()

			// Create a hasher to reuse between invocations
			keccak512 := makeHasher(sha3.NewKeccak512())

			// Calculate the data segment this thread should generate
			batch := uint32((size + hashBytes*uint64(threads) - 1) / (hashBytes * uint64(threads)))
			first := uint32(id) * batch
			limit := first + batch
			if limit > uint32(size/hashBytes) {
				limit = uint32(size / hashBytes)
			}
			// Calculate the dataset segment
			percent := uint32(size / hashBytes / 100)
			if percent == 0 {
				percent = 1
			}
			for index := first; index < limit; index++ {
				item := generateDatasetItem(cache, index, keccak512)
				if swapped {
					swap(item)
				}
				copy(dataset[index*hashBytes:], item)

				if status := atomic.AddUint32(&progress, 1); status%percent == 0 {
					logger.Info("Generating DAG in progress", "percentage", uint64(status*100)/(size/hashBytes), "elapsed", common.PrettyDuration(time.Since(start)))
				}
			}
		}(i)
	}
	// Wait for all the generators to finish and return
	pend.Wait()
}

// hashimoto aggregates data from the full dataset in order to produce our final
// value for a particular header hash and nonce.
func hashimoto(hash []byte, nonce uint64, size uint64, lookup func(index uint32) []uint32) ([]byte, []byte) {
	// Calculate the number of theoretical rows (we use one buffer nonetheless)
	rows := uint32(size / mixBytes)

	// Combine header+nonce into a 64 byte seed
	seed := make([]byte, 40)
	copy(seed, hash)
	binary.LittleEndian.PutUint64(seed[32:], nonce)

	seed = crypto.Keccak512(seed)
	seedHead := binary.LittleEndian.Uint32(seed)

	// Start the mix with replicated seed
	mix := make([]uint32, mixBytes/4)
	for i := 0; i < len(mix); i++ {
		mix[i] = binary.LittleEndian.Uint32(seed[i%16*4:])
	}
	// Mix in random dataset nodes
	temp := make([]uint32, len(mix))

	for i := 0; i < loopAccesses; i++ {
		parent := fnv(uint32(i)^seedHead, mix[i%len(mix)]) % rows
		for j := uint32(0); j < mixBytes/hashBytes; j++ {
			copy(temp[j*hashWords:], lookup(2*parent+j))
		}
		fnvHash(mix, temp)
	}
	// Compress mix
	for i := 0; i < len(mix); i += 4 {
		mix[i/4] = fnv(fnv(fnv(mix[i], mix[i+1]), mix[i+2]), mix[i+3])
	}
	mix = mix[:len(mix)/4]

	digest := make([]byte, common.HashLength)
	for i, val := range mix {
		binary.LittleEndian.PutUint32(digest[i*4:], val)
	}
	return digest, crypto.Keccak256(append(seed, digest...))
}

// hashimotoLight aggregates data from the full dataset (using only a small
// in-memory cache) in order to produce our final value for a particular header
// hash and nonce.
func hashimotoLight(size uint64, cache []uint32, hash []byte, nonce uint64) ([]byte, []byte) {
	keccak512 := makeHasher(sha3.NewKeccak512())

	lookup := func(index uint32) []uint32 {
		rawData := generateDatasetItem(cache, index, keccak512)

		data := make([]uint32, len(rawData)/4)
		for i := 0; i < len(data); i++ {
			data[i] = binary.LittleEndian.Uint32(rawData[i*4:])
		}
		return data
	}
	return hashimoto(hash, nonce, size, lookup)
}

// hashimotoFull aggregates data from the full dataset (using the full in-memory
// dataset) in order to produce our final value for a particular header hash and
// nonce.
func hashimotoFull(dataset []uint32, hash []byte, nonce uint64) ([]byte, []byte) {
	lookup := func(index uint32) []uint32 {
		offset := index * hashWords
		return dataset[offset : offset+hashWords]
	}
	return hashimoto(hash, nonce, uint64(len(dataset))*4, lookup)
}

const MaxEpoch = 2048

// datasetSizes is a lookup table for the aquahash dataset size for the first 2048
// epochs (i.e. 61440000 blocks).
var datasetSizes = [MaxEpoch]uint64{
	1073739904, 1082130304, 1090514816, 1098906752, 1107293056,
	1115684224, 1124070016, 1132461952, 1140849536, 1149232768,
	1157627776, 1166013824, 1174404736, 1182786944, 1191180416,
	1199568512, 1207958912, 1216345216, 1224732032, 1233124736,
	1241513344, 1249902464, 1258290304, 1266673792, 1275067264,
	1283453312, 1291844992, 1300234112, 1308619904, 1317010048,
	1325397376, 1333787776, 1342176128, 1350561664, 1358954368,
	1367339392, 1375731584, 1384118144, 1392507008, 1400897408,
	1409284736, 1417673344, 1426062464, 1434451072, 1442839168,
	1451229056, 1459615616, 1468006016, 1476394112, 1484782976,
	1493171584, 1501559168, 1509948032, 1518337664, 1526726528,
	1535114624, 1543503488, 1551892096, 1560278656, 1568669056,
	1577056384, 1585446272, 1593831296, 1602219392, 1610610304,
	1619000192, 1627386752, 1635773824, 1644164224, 1652555648,
	1660943488, 1669332608, 1677721216, 1686109312, 1694497664,
	1702886272, 1711274624, 1719661184, 1728047744, 1736434816,
	1744829056, 1753218944, 1761606272, 1769995904, 1778382464,
	1786772864, 1795157888, 1803550592, 1811937664, 1820327552,
	1828711552, 1837102976, 1845488768, 1853879936, 1862269312,
	1870656896, 1879048064, 1887431552, 1895825024, 1904212096,
	1912601216, 1920988544, 1929379456, 1937765504, 1946156672,
	1954543232, 1962932096, 1971321728, 1979707264, 1988093056,
	1996487552, 2004874624, 2013262208, 2021653888, 2030039936,
	2038430848, 2046819968, 2055208576, 2063596672, 2071981952,
	2080373632, 2088762752, 2097149056, 2105539712, 2113928576,
	2122315136, 2130700672, 2139092608, 2147483264, 2155872128,
	2164257664, 2172642176, 2181035392, 2189426048, 2197814912,
	2206203008, 2214587264, 2222979712, 2231367808, 2239758208,
	2248145024, 2256527744, 2264922752, 2273312128, 2281701248,
	2290086272, 2298476672, 2306867072, 2315251072, 2323639168,
	2332032128, 2340420224, 2348808064, 2357196416, 2365580416,
	2373966976, 2382363008, 2390748544, 2399139968, 2407530368,
	2415918976, 2424307328, 2432695424, 2441084288, 2449472384,
	2457861248, 2466247808, 2474637184, 2483026816, 2491414144,
	2499803776, 2508191872, 2516582272, 2524970368, 2533359232,
	2541743488, 2550134144, 2558525056, 2566913408, 2575301504,
	2583686528, 2592073856, 2600467328, 2608856192, 2617240448,
	2625631616, 2634022016, 2642407552, 2650796416, 2659188352,
	2667574912, 2675965312, 2684352896, 2692738688, 2701130624,
	2709518464, 2717907328, 2726293376, 2734685056, 2743073152,
	2751462016, 2759851648, 2768232832, 2776625536, 2785017728,
	2793401984, 2801794432, 2810182016, 2818571648, 2826959488,
	2835349376, 2843734144, 2852121472, 2860514432, 2868900992,
	2877286784, 2885676928, 2894069632, 2902451584, 2910843008,
	2919234688, 2927622784, 2936011648, 2944400768, 2952789376,
	2961177728, 2969565568, 2977951616, 2986338944, 2994731392,
	3003120256, 3011508352, 3019895936, 3028287104, 3036675968,
	3045063808, 3053452928, 3061837696, 3070228352, 3078615424,
	3087003776, 3095394944, 3103782272, 3112173184, 3120562048,
	3128944768, 3137339264, 3145725056, 3154109312, 3162505088,
	3170893184, 3179280256, 3187669376, 3196056704, 3204445568,
	3212836736, 3221224064, 3229612928, 3238002304, 3246391168,
	3254778496, 3263165824, 3271556224, 3279944576, 3288332416,
	3296719232, 3305110912, 3313500032, 3321887104, 3330273152,
	3338658944, 3347053184, 3355440512, 3363827072, 3372220288,
	3380608384, 3388997504, 3397384576, 3405774208, 3414163072,
	3422551936, 3430937984, 3439328384, 3447714176, 3456104576,
	3464493952, 3472883584, 3481268864, 3489655168, 3498048896,
	3506434432, 3514826368, 3523213952, 3531603584, 3539987072,
	3548380288, 3556763264, 3565157248, 3573545344, 3581934464,
	3590324096, 3598712704, 3607098752, 3615488384, 3623877248,
	3632265856, 3640646528, 3649043584, 3657430144, 3665821568,
	3674207872, 3682597504, 3690984832, 3699367808, 3707764352,
	3716152448, 3724541056, 3732925568, 3741318016, 3749706368,
	3758091136, 3766481536, 3774872704, 3783260032, 3791650432,
	3800036224, 3808427648, 3816815488, 3825204608, 3833592704,
	3841981568, 3850370432, 3858755968, 3867147904, 3875536256,
	3883920512, 3892313728, 3900702592, 3909087872, 3917478784,
	3925868416, 3934256512, 3942645376, 3951032192, 3959422336,
	3967809152, 3976200064, 3984588416, 3992974976, 4001363584,
	4009751168, 4018141312, 4026530432, 4034911616, 4043308928,
	4051695488, 4060084352, 4068472448, 4076862848, 4085249408,
	4093640576, 4102028416, 4110413696, 4118805632, 4127194496,
	4135583104, 4143971968, 4152360832, 4160746112, 4169135744,
	4177525888, 4185912704, 4194303616, 4202691968, 4211076736,
	4219463552, 4227855488, 4236246656, 4244633728, 4253022848,
	4261412224, 4269799808, 4278184832, 4286578048, 4294962304,
	4303349632, 4311743104, 4320130432, 4328521088, 4336909184,
	4345295488, 4353687424, 4362073472, 4370458496, 4378852736,
	4387238528, 4395630208, 4404019072, 4412407424, 4420790656,
	4429182848, 4437571456, 4445962112, 4454344064, 4462738048,
	4471119232, 4479516544, 4487904128, 4496289664, 4504682368,
	4513068416, 4521459584, 4529846144, 4538232704, 4546619776,
	4555010176, 4563402112, 4571790208, 4580174464, 4588567936,
	4596957056, 4605344896, 4613734016, 4622119808, 4630511488,
	4638898816, 4647287936, 4655675264, 4664065664, 4672451968,
	4680842624, 4689231488, 4697620352, 4706007424, 4714397056,
	4722786176, 4731173248, 4739562368, 4747951744, 4756340608,
	4764727936, 4773114496, 4781504384, 4789894784, 4798283648,
	4806667648, 4815059584, 4823449472, 4831835776, 4840226176,
	4848612224, 4857003392, 4865391488, 4873780096, 4882169728,
	4890557312, 4898946944, 4907333248, 4915722368, 4924110976,
	4932499328, 4940889728, 4949276032, 4957666432, 4966054784,
	4974438016, 4982831488, 4991221376, 4999607168, 5007998848,
	5016386432, 5024763776, 5033164672, 5041544576, 5049941888,
	5058329728, 5066717056, 5075107456, 5083494272, 5091883904,
	5100273536, 5108662144, 5117048192, 5125436032, 5133827456,
	5142215296, 5150605184, 5158993024, 5167382144, 5175769472,
	5184157568, 5192543872, 5200936064, 5209324928, 5217711232,
	5226102656, 5234490496, 5242877312, 5251263872, 5259654016,
	5268040832, 5276434304, 5284819328, 5293209728, 5301598592,
	5309986688, 5318374784, 5326764416, 5335151488, 5343542144,
	5351929472, 5360319872, 5368706944, 5377096576, 5385484928,
	5393871232, 5402263424, 5410650496, 5419040384, 5427426944,
	5435816576, 5444205952, 5452594816, 5460981376, 5469367936,
	5477760896, 5486148736, 5494536832, 5502925952, 5511315328,
	5519703424, 5528089984, 5536481152, 5544869504, 5553256064,
	5561645696, 5570032768, 5578423936, 5586811264, 5595193216,
	5603585408, 5611972736, 5620366208, 5628750464, 5637143936,
	5645528192, 5653921408, 5662310272, 5670694784, 5679082624,
	5687474048, 5695864448, 5704251008, 5712641408, 5721030272,
	5729416832, 5737806208, 5746194304, 5754583936, 5762969984,
	5771358592, 5779748224, 5788137856, 5796527488, 5804911232,
	5813300608, 5821692544, 5830082176, 5838468992, 5846855552,
	5855247488, 5863636096, 5872024448, 5880411008, 5888799872,
	5897186432, 5905576832, 5913966976, 5922352768, 5930744704,
	5939132288, 5947522432, 5955911296, 5964299392, 5972688256,
	5981074304, 5989465472, 5997851008, 6006241408, 6014627968,
	6023015552, 6031408256, 6039796096, 6048185216, 6056574848,
	6064963456, 6073351808, 6081736064, 6090128768, 6098517632,
	6106906496, 6115289216, 6123680896, 6132070016, 6140459648,
	6148849024, 6157237376, 6165624704, 6174009728, 6182403712,
	6190792064, 6199176064, 6207569792, 6215952256, 6224345216,
	6232732544, 6241124224, 6249510272, 6257899136, 6266287744,
	6274676864, 6283065728, 6291454336, 6299843456, 6308232064,
	6316620928, 6325006208, 6333395584, 6341784704, 6350174848,
	6358562176, 6366951296, 6375337856, 6383729536, 6392119168,
	6400504192, 6408895616, 6417283456, 6425673344, 6434059136,
	6442444672, 6450837376, 6459223424, 6467613056, 6476004224,
	6484393088, 6492781952, 6501170048, 6509555072, 6517947008,
	6526336384, 6534725504, 6543112832, 6551500672, 6559888768,
	6568278656, 6576662912, 6585055616, 6593443456, 6601834112,
	6610219648, 6618610304, 6626999168, 6635385472, 6643777408,
	6652164224, 6660552832, 6668941952, 6677330048, 6685719424,
	6694107776, 6702493568, 6710882176, 6719274112, 6727662976,
	6736052096, 6744437632, 6752825984, 6761213824, 6769604224,
	6777993856, 6786383488, 6794770816, 6803158144, 6811549312,
	6819937664, 6828326528, 6836706176, 6845101696, 6853491328,
	6861880448, 6870269312, 6878655104, 6887046272, 6895433344,
	6903822208, 6912212864, 6920596864, 6928988288, 6937377152,
	6945764992, 6954149248, 6962544256, 6970928768, 6979317376,
	6987709312, 6996093824, 7004487296, 7012875392, 7021258624,
	7029652352, 7038038912, 7046427776, 7054818944, 7063207808,
	7071595136, 7079980928, 7088372608, 7096759424, 7105149824,
	7113536896, 7121928064, 7130315392, 7138699648, 7147092352,
	7155479168, 7163865728, 7172249984, 7180648064, 7189036672,
	7197424768, 7205810816, 7214196608, 7222589824, 7230975104,
	7239367552, 7247755904, 7256145536, 7264533376, 7272921472,
	7281308032, 7289694848, 7298088832, 7306471808, 7314864512,
	7323253888, 7331643008, 7340029568, 7348419712, 7356808832,
	7365196672, 7373585792, 7381973888, 7390362752, 7398750592,
	7407138944, 7415528576, 7423915648, 7432302208, 7440690304,
	7449080192, 7457472128, 7465860992, 7474249088, 7482635648,
	7491023744, 7499412608, 7507803008, 7516192384, 7524579968,
	7532967296, 7541358464, 7549745792, 7558134656, 7566524032,
	7574912896, 7583300992, 7591690112, 7600075136, 7608466816,
	7616854912, 7625244544, 7633629824, 7642020992, 7650410368,
	7658794112, 7667187328, 7675574912, 7683961984, 7692349568,
	7700739712, 7709130368, 7717519232, 7725905536, 7734295424,
	7742683264, 7751069056, 7759457408, 7767849088, 7776238208,
	7784626816, 7793014912, 7801405312, 7809792128, 7818179968,
	7826571136, 7834957184, 7843347328, 7851732352, 7860124544,
	7868512384, 7876902016, 7885287808, 7893679744, 7902067072,
	7910455936, 7918844288, 7927230848, 7935622784, 7944009344,
	7952400256, 7960786048, 7969176704, 7977565312, 7985953408,
	7994339968, 8002730368, 8011119488, 8019508096, 8027896192,
	8036285056, 8044674688, 8053062272, 8061448832, 8069838464,
	8078227328, 8086616704, 8095006592, 8103393664, 8111783552,
	8120171392, 8128560256, 8136949376, 8145336704, 8153726848,
	8162114944, 8170503296, 8178891904, 8187280768, 8195669632,
	8204058496, 8212444544, 8220834176, 8229222272, 8237612672,
	8246000768, 8254389376, 8262775168, 8271167104, 8279553664,
	8287944064, 8296333184, 8304715136, 8313108352, 8321497984,
	8329885568, 8338274432, 8346663296, 8355052928, 8363441536,
	8371828352, 8380217984, 8388606592, 8396996224, 8405384576,
	8413772672, 8422161536, 8430549376, 8438939008, 8447326592,
	8455715456, 8464104832, 8472492928, 8480882048, 8489270656,
	8497659776, 8506045312, 8514434944, 8522823808, 8531208832,
	8539602304, 8547990656, 8556378752, 8564768384, 8573154176,
	8581542784, 8589933952, 8598322816, 8606705024, 8615099264,
	8623487872, 8631876992, 8640264064, 8648653952, 8657040256,
	8665430656, 8673820544, 8682209152, 8690592128, 8698977152,
	8707374464, 8715763328, 8724151424, 8732540032, 8740928384,
	8749315712, 8757704576, 8766089344, 8774480768, 8782871936,
	8791260032, 8799645824, 8808034432, 8816426368, 8824812928,
	8833199488, 8841591424, 8849976448, 8858366336, 8866757248,
	8875147136, 8883532928, 8891923328, 8900306816, 8908700288,
	8917088384, 8925478784, 8933867392, 8942250368, 8950644608,
	8959032704, 8967420544, 8975809664, 8984197504, 8992584064,
	9000976256, 9009362048, 9017752448, 9026141312, 9034530688,
	9042917504, 9051307904, 9059694208, 9068084864, 9076471424,
	9084861824, 9093250688, 9101638528, 9110027648, 9118416512,
	9126803584, 9135188096, 9143581312, 9151969664, 9160356224,
	9168747136, 9177134464, 9185525632, 9193910144, 9202302848,
	9210690688, 9219079552, 9227465344, 9235854464, 9244244864,
	9252633472, 9261021824, 9269411456, 9277799296, 9286188928,
	9294574208, 9302965888, 9311351936, 9319740032, 9328131968,
	9336516736, 9344907392, 9353296768, 9361685888, 9370074752,
	9378463616, 9386849408, 9395239808, 9403629184, 9412016512,
	9420405376, 9428795008, 9437181568, 9445570688, 9453960832,
	9462346624, 9470738048, 9479121536, 9487515008, 9495903616,
	9504289664, 9512678528, 9521067904, 9529456256, 9537843584,
	9546233728, 9554621312, 9563011456, 9571398784, 9579788672,
	9588178304, 9596567168, 9604954496, 9613343104, 9621732992,
	9630121856, 9638508416, 9646898816, 9655283584, 9663675776,
	9672061312, 9680449664, 9688840064, 9697230464, 9705617536,
	9714003584, 9722393984, 9730772608, 9739172224, 9747561088,
	9755945344, 9764338816, 9772726144, 9781116544, 9789503872,
	9797892992, 9806282624, 9814670464, 9823056512, 9831439232,
	9839833984, 9848224384, 9856613504, 9865000576, 9873391232,
	9881772416, 9890162816, 9898556288, 9906940544, 9915333248,
	9923721088, 9932108672, 9940496512, 9948888448, 9957276544,
	9965666176, 9974048384, 9982441088, 9990830464, 9999219584,
	10007602816, 10015996544, 10024385152, 10032774016, 10041163648,
	10049548928, 10057940096, 10066329472, 10074717824, 10083105152,
	10091495296, 10099878784, 10108272256, 10116660608, 10125049216,
	10133437312, 10141825664, 10150213504, 10158601088, 10166991232,
	10175378816, 10183766144, 10192157312, 10200545408, 10208935552,
	10217322112, 10225712768, 10234099328, 10242489472, 10250876032,
	10259264896, 10267656064, 10276042624, 10284429184, 10292820352,
	10301209472, 10309598848, 10317987712, 10326375296, 10334763392,
	10343153536, 10351541632, 10359930752, 10368318592, 10376707456,
	10385096576, 10393484672, 10401867136, 10410262144, 10418647424,
	10427039104, 10435425664, 10443810176, 10452203648, 10460589952,
	10468982144, 10477369472, 10485759104, 10494147712, 10502533504,
	10510923392, 10519313536, 10527702656, 10536091264, 10544478592,
	10552867712, 10561255808, 10569642368, 10578032768, 10586423168,
	10594805632, 10603200128, 10611588992, 10619976064, 10628361344,
	10636754048, 10645143424, 10653531776, 10661920384, 10670307968,
	10678696832, 10687086464, 10695475072, 10703863168, 10712246144,
	10720639616, 10729026688, 10737414784, 10745806208, 10754190976,
	10762581376, 10770971264, 10779356288, 10787747456, 10796135552,
	10804525184, 10812915584, 10821301888, 10829692288, 10838078336,
	10846469248, 10854858368, 10863247232, 10871631488, 10880023424,
	10888412032, 10896799616, 10905188992, 10913574016, 10921964672,
	10930352768, 10938742912, 10947132544, 10955518592, 10963909504,
	10972298368, 10980687488, 10989074816, 10997462912, 11005851776,
	11014241152, 11022627712, 11031017344, 11039403904, 11047793024,
	11056184704, 11064570752, 11072960896, 11081343872, 11089737856,
	11098128256, 11106514816, 11114904448, 11123293568, 11131680128,
	11140065152, 11148458368, 11156845696, 11165236864, 11173624192,
	11182013824, 11190402688, 11198790784, 11207179136, 11215568768,
	11223957376, 11232345728, 11240734592, 11249122688, 11257511296,
	11265899648, 11274285952, 11282675584, 11291065472, 11299452544,
	11307842432, 11316231296, 11324616832, 11333009024, 11341395584,
	11349782656, 11358172288, 11366560384, 11374950016, 11383339648,
	11391721856, 11400117376, 11408504192, 11416893568, 11425283456,
	11433671552, 11442061184, 11450444672, 11458837888, 11467226752,
	11475611776, 11484003968, 11492392064, 11500780672, 11509169024,
	11517550976, 11525944448, 11534335616, 11542724224, 11551111808,
	11559500672, 11567890304, 11576277376, 11584667008, 11593056128,
	11601443456, 11609830016, 11618221952, 11626607488, 11634995072,
	11643387776, 11651775104, 11660161664, 11668552576, 11676940928,
	11685330304, 11693718656, 11702106496, 11710496128, 11718882688,
	11727273088, 11735660416, 11744050048, 11752437376, 11760824704,
	11769216128, 11777604736, 11785991296, 11794381952, 11802770048,
	11811157888, 11819548544, 11827932544, 11836324736, 11844713344,
	11853100928, 11861486464, 11869879936, 11878268032, 11886656896,
	11895044992, 11903433088, 11911822976, 11920210816, 11928600448,
	11936987264, 11945375872, 11953761152, 11962151296, 11970543488,
	11978928512, 11987320448, 11995708288, 12004095104, 12012486272,
	12020875136, 12029255552, 12037652096, 12046039168, 12054429568,
	12062813824, 12071206528, 12079594624, 12087983744, 12096371072,
	12104759936, 12113147264, 12121534592, 12129924992, 12138314624,
	12146703232, 12155091584, 12163481216, 12171864704, 12180255872,
	12188643968, 12197034112, 12205424512, 12213811328, 12222199424,
	12230590336, 12238977664, 12247365248, 12255755392, 12264143488,
	12272531584, 12280920448, 12289309568, 12297694592, 12306086528,
	12314475392, 12322865024, 12331253632, 12339640448, 12348029312,
	12356418944, 12364805248, 12373196672, 12381580928, 12389969024,
	12398357632, 12406750592, 12415138432, 12423527552, 12431916416,
	12440304512, 12448692352, 12457081216, 12465467776, 12473859968,
	12482245504, 12490636672, 12499025536, 12507411584, 12515801728,
	12524190592, 12532577152, 12540966272, 12549354368, 12557743232,
	12566129536, 12574523264, 12582911872, 12591299456, 12599688064,
	12608074624, 12616463488, 12624845696, 12633239936, 12641631616,
	12650019968, 12658407296, 12666795136, 12675183232, 12683574656,
	12691960192, 12700350592, 12708740224, 12717128576, 12725515904,
	12733906816, 12742295168, 12750680192, 12759071872, 12767460736,
	12775848832, 12784236928, 12792626816, 12801014656, 12809404288,
	12817789312, 12826181504, 12834568832, 12842954624, 12851345792,
	12859732352, 12868122496, 12876512128, 12884901248, 12893289088,
	12901672832, 12910067584, 12918455168, 12926842496, 12935232896,
	12943620736, 12952009856, 12960396928, 12968786816, 12977176192,
	12985563776, 12993951104, 13002341504, 13010730368, 13019115392,
	13027506304, 13035895168, 13044272512, 13052673152, 13061062528,
	13069446272, 13077838976, 13086227072, 13094613632, 13103000192,
	13111393664, 13119782528, 13128157568, 13136559232, 13144945024,
	13153329536, 13161724288, 13170111872, 13178502784, 13186884736,
	13195279744, 13203667072, 13212057472, 13220445824, 13228832128,
	13237221248, 13245610624, 13254000512, 13262388352, 13270777472,
	13279166336, 13287553408, 13295943296, 13304331904, 13312719488,
	13321108096, 13329494656, 13337885824, 13346274944, 13354663808,
	13363051136, 13371439232, 13379825024, 13388210816, 13396605056,
	13404995456, 13413380224, 13421771392, 13430159744, 13438546048,
	13446937216, 13455326848, 13463708288, 13472103808, 13480492672,
	13488875648, 13497269888, 13505657728, 13514045312, 13522435712,
	13530824576, 13539210112, 13547599232, 13555989376, 13564379008,
	13572766336, 13581154432, 13589544832, 13597932928, 13606320512,
	13614710656, 13623097472, 13631477632, 13639874944, 13648264064,
	13656652928, 13665041792, 13673430656, 13681818496, 13690207616,
	13698595712, 13706982272, 13715373184, 13723762048, 13732150144,
	13740536704, 13748926592, 13757316224, 13765700992, 13774090112,
	13782477952, 13790869376, 13799259008, 13807647872, 13816036736,
	13824425344, 13832814208, 13841202304, 13849591424, 13857978752,
	13866368896, 13874754688, 13883145344, 13891533184, 13899919232,
	13908311168, 13916692096, 13925085056, 13933473152, 13941866368,
	13950253696, 13958643584, 13967032192, 13975417216, 13983807616,
	13992197504, 14000582272, 14008973696, 14017363072, 14025752192,
	14034137984, 14042528384, 14050918016, 14059301504, 14067691648,
	14076083584, 14084470144, 14092852352, 14101249664, 14109635968,
	14118024832, 14126407552, 14134804352, 14143188608, 14151577984,
	14159968384, 14168357248, 14176741504, 14185127296, 14193521024,
	14201911424, 14210301824, 14218685056, 14227067264, 14235467392,
	14243855488, 14252243072, 14260630144, 14269021568, 14277409408,
	14285799296, 14294187904, 14302571392, 14310961792, 14319353728,
	14327738752, 14336130944, 14344518784, 14352906368, 14361296512,
	14369685376, 14378071424, 14386462592, 14394848128, 14403230848,
	14411627392, 14420013952, 14428402304, 14436793472, 14445181568,
	14453569664, 14461959808, 14470347904, 14478737024, 14487122816,
	14495511424, 14503901824, 14512291712, 14520677504, 14529064832,
	14537456768, 14545845632, 14554234496, 14562618496, 14571011456,
	14579398784, 14587789184, 14596172672, 14604564608, 14612953984,
	14621341312, 14629724288, 14638120832, 14646503296, 14654897536,
	14663284864, 14671675264, 14680061056, 14688447616, 14696835968,
	14705228416, 14713616768, 14722003328, 14730392192, 14738784128,
	14747172736, 14755561088, 14763947648, 14772336512, 14780725376,
	14789110144, 14797499776, 14805892736, 14814276992, 14822670208,
	14831056256, 14839444352, 14847836032, 14856222848, 14864612992,
	14872997504, 14881388672, 14889775744, 14898165376, 14906553472,
	14914944896, 14923329664, 14931721856, 14940109696, 14948497024,
	14956887424, 14965276544, 14973663616, 14982053248, 14990439808,
	14998830976, 15007216768, 15015605888, 15023995264, 15032385152,
	15040768384, 15049154944, 15057549184, 15065939072, 15074328448,
	15082715008, 15091104128, 15099493504, 15107879296, 15116269184,
	15124659584, 15133042304, 15141431936, 15149824384, 15158214272,
	15166602368, 15174991232, 15183378304, 15191760512, 15200154496,
	15208542592, 15216931712, 15225323392, 15233708416, 15242098048,
	15250489216, 15258875264, 15267265408, 15275654528, 15284043136,
	15292431488, 15300819584, 15309208192, 15317596544, 15325986176,
	15334374784, 15342763648, 15351151744, 15359540608, 15367929728,
	15376318336, 15384706432, 15393092992, 15401481856, 15409869952,
	15418258816, 15426649984, 15435037568, 15443425664, 15451815296,
	15460203392, 15468589184, 15476979328, 15485369216, 15493755776,
	15502146944, 15510534272, 15518924416, 15527311232, 15535699072,
	15544089472, 15552478336, 15560866688, 15569254528, 15577642624,
	15586031488, 15594419072, 15602809472, 15611199104, 15619586432,
	15627975296, 15636364928, 15644753792, 15653141888, 15661529216,
	15669918848, 15678305152, 15686696576, 15695083136, 15703474048,
	15711861632, 15720251264, 15728636288, 15737027456, 15745417088,
	15753804928, 15762194048, 15770582656, 15778971008, 15787358336,
	15795747712, 15804132224, 15812523392, 15820909696, 15829300096,
	15837691264, 15846071936, 15854466944, 15862855808, 15871244672,
	15879634816, 15888020608, 15896409728, 15904799104, 15913185152,
	15921577088, 15929966464, 15938354816, 15946743424, 15955129472,
	15963519872, 15971907968, 15980296064, 15988684928, 15997073024,
	16005460864, 16013851264, 16022241152, 16030629248, 16039012736,
	16047406976, 16055794816, 16064181376, 16072571264, 16080957824,
	16089346688, 16097737856, 16106125184, 16114514816, 16122904192,
	16131292544, 16139678848, 16148066944, 16156453504, 16164839552,
	16173236096, 16181623424, 16190012032, 16198401152, 16206790528,
	16215177344, 16223567744, 16231956352, 16240344704, 16248731008,
	16257117824, 16265504384, 16273898624, 16282281856, 16290668672,
	16299064192, 16307449216, 16315842176, 16324230016, 16332613504,
	16341006464, 16349394304, 16357783168, 16366172288, 16374561664,
	16382951296, 16391337856, 16399726208, 16408116352, 16416505472,
	16424892032, 16433282176, 16441668224, 16450058624, 16458448768,
	16466836864, 16475224448, 16483613056, 16492001408, 16500391808,
	16508779648, 16517166976, 16525555328, 16533944192, 16542330752,
	16550719616, 16559110528, 16567497088, 16575888512, 16584274816,
	16592665472, 16601051008, 16609442944, 16617832064, 16626218624,
	16634607488, 16642996096, 16651385728, 16659773824, 16668163712,
	16676552576, 16684938112, 16693328768, 16701718144, 16710095488,
	16718492288, 16726883968, 16735272832, 16743661184, 16752049792,
	16760436608, 16768827008, 16777214336, 16785599104, 16793992832,
	16802381696, 16810768768, 16819151744, 16827542656, 16835934848,
	16844323712, 16852711552, 16861101952, 16869489536, 16877876864,
	16886265728, 16894653056, 16903044736, 16911431296, 16919821696,
	16928207488, 16936592768, 16944987776, 16953375616, 16961763968,
	16970152832, 16978540928, 16986929536, 16995319168, 17003704448,
	17012096896, 17020481152, 17028870784, 17037262208, 17045649536,
	17054039936, 17062426496, 17070814336, 17079205504, 17087592064,
	17095978112, 17104369024, 17112759424, 17121147776, 17129536384,
	17137926016, 17146314368, 17154700928, 17163089792, 17171480192,
	17179864192, 17188256896, 17196644992, 17205033856, 17213423488,
	17221811072, 17230198912, 17238588032, 17246976896, 17255360384,
	17263754624, 17272143232, 17280530048, 17288918912, 17297309312,
	17305696384, 17314085504, 17322475136, 17330863744, 17339252096,
	17347640192, 17356026496, 17364413824, 17372796544, 17381190016,
	17389583488, 17397972608, 17406360704, 17414748544, 17423135872,
	17431527296, 17439915904, 17448303232, 17456691584, 17465081728,
	17473468288, 17481857408, 17490247552, 17498635904, 17507022464,
	17515409024, 17523801728, 17532189824, 17540577664, 17548966016,
	17557353344, 17565741184, 17574131584, 17582519168, 17590907008,
	17599296128, 17607687808, 17616076672, 17624455808, 17632852352,
	17641238656, 17649630848, 17658018944, 17666403968, 17674794112,
	17683178368, 17691573376, 17699962496, 17708350592, 17716739968,
	17725126528, 17733517184, 17741898112, 17750293888, 17758673024,
	17767070336, 17775458432, 17783848832, 17792236928, 17800625536,
	17809012352, 17817402752, 17825785984, 17834178944, 17842563968,
	17850955648, 17859344512, 17867732864, 17876119424, 17884511872,
	17892900224, 17901287296, 17909677696, 17918058112, 17926451072,
	17934843776, 17943230848, 17951609216, 17960008576, 17968397696,
	17976784256, 17985175424, 17993564032, 18001952128, 18010339712,
	18018728576, 18027116672, 18035503232, 18043894144, 18052283264,
	18060672128, 18069056384, 18077449856, 18085837184, 18094225792,
	18102613376, 18111004544, 18119388544, 18127781248, 18136170368,
	18144558976, 18152947328, 18161336192, 18169724288, 18178108544,
	18186498944, 18194886784, 18203275648, 18211666048, 18220048768,
	18228444544, 18236833408, 18245220736}

// cacheSizes is a lookup table for the aquahash verification cache size for the
// first 2048 epochs (i.e. 61440000 blocks).
var cacheSizes = [maxEpoch]uint64{
	16776896, 16907456, 17039296, 17170112, 17301056, 17432512, 17563072,
	17693888, 17824192, 17955904, 18087488, 18218176, 18349504, 18481088,
	18611392, 18742336, 18874304, 19004224, 19135936, 19267264, 19398208,
	19529408, 19660096, 19791424, 19922752, 20053952, 20184896, 20315968,
	20446912, 20576576, 20709184, 20840384, 20971072, 21102272, 21233216,
	21364544, 21494848, 21626816, 21757376, 21887552, 22019392, 22151104,
	22281536, 22412224, 22543936, 22675264, 22806464, 22935872, 23068096,
	23198272, 23330752, 23459008, 23592512, 23723968, 23854912, 23986112,
	24116672, 24247616, 24378688, 24509504, 24640832, 24772544, 24903488,
	25034432, 25165376, 25296704, 25427392, 25558592, 25690048, 25820096,
	25951936, 26081728, 26214208, 26345024, 26476096, 26606656, 26737472,
	26869184, 26998208, 27131584, 27262528, 27393728, 27523904, 27655744,
	27786688, 27917888, 28049344, 28179904, 28311488, 28441792, 28573504,
	28700864, 28835648, 28966208, 29096768, 29228608, 29359808, 29490752,
	29621824, 29752256, 29882816, 30014912, 30144448, 30273728, 30406976,
	30538432, 30670784, 30799936, 30932672, 31063744, 31195072, 31325248,
	31456192, 31588288, 31719232, 31850432, 31981504, 32110784, 32243392,
	32372672, 32505664, 32636608, 32767808, 32897344, 33029824, 33160768,
	33289664, 33423296, 33554368, 33683648, 33816512, 33947456, 34076992,
	34208704, 34340032, 34471744, 34600256, 34734016, 34864576, 34993984,
	35127104, 35258176, 35386688, 35518528, 35650624, 35782336, 35910976,
	36044608, 36175808, 36305728, 36436672, 36568384, 36699968, 36830656,
	36961984, 37093312, 37223488, 37355072, 37486528, 37617472, 37747904,
	37879232, 38009792, 38141888, 38272448, 38403392, 38535104, 38660672,
	38795584, 38925632, 39059264, 39190336, 39320768, 39452096, 39581632,
	39713984, 39844928, 39974848, 40107968, 40238144, 40367168, 40500032,
	40631744, 40762816, 40894144, 41023552, 41155904, 41286208, 41418304,
	41547712, 41680448, 41811904, 41942848, 42073792, 42204992, 42334912,
	42467008, 42597824, 42729152, 42860096, 42991552, 43122368, 43253696,
	43382848, 43515712, 43646912, 43777088, 43907648, 44039104, 44170432,
	44302144, 44433344, 44564288, 44694976, 44825152, 44956864, 45088448,
	45219008, 45350464, 45481024, 45612608, 45744064, 45874496, 46006208,
	46136768, 46267712, 46399424, 46529344, 46660672, 46791488, 46923328,
	47053504, 47185856, 47316928, 47447872, 47579072, 47710144, 47839936,
	47971648, 48103232, 48234176, 48365248, 48496192, 48627136, 48757312,
	48889664, 49020736, 49149248, 49283008, 49413824, 49545152, 49675712,
	49807168, 49938368, 50069056, 50200256, 50331584, 50462656, 50593472,
	50724032, 50853952, 50986048, 51117632, 51248576, 51379904, 51510848,
	51641792, 51773248, 51903296, 52035136, 52164032, 52297664, 52427968,
	52557376, 52690112, 52821952, 52952896, 53081536, 53213504, 53344576,
	53475776, 53608384, 53738816, 53870528, 54000832, 54131776, 54263744,
	54394688, 54525248, 54655936, 54787904, 54918592, 55049152, 55181248,
	55312064, 55442752, 55574336, 55705024, 55836224, 55967168, 56097856,
	56228672, 56358592, 56490176, 56621888, 56753728, 56884928, 57015488,
	57146816, 57278272, 57409216, 57540416, 57671104, 57802432, 57933632,
	58064576, 58195264, 58326976, 58457408, 58588864, 58720192, 58849984,
	58981696, 59113024, 59243456, 59375552, 59506624, 59637568, 59768512,
	59897792, 60030016, 60161984, 60293056, 60423872, 60554432, 60683968,
	60817216, 60948032, 61079488, 61209664, 61341376, 61471936, 61602752,
	61733696, 61865792, 61996736, 62127808, 62259136, 62389568, 62520512,
	62651584, 62781632, 62910784, 63045056, 63176128, 63307072, 63438656,
	63569216, 63700928, 63831616, 63960896, 64093888, 64225088, 64355392,
	64486976, 64617664, 64748608, 64879424, 65009216, 65142464, 65273792,
	65402816, 65535424, 65666752, 65797696, 65927744, 66060224, 66191296,
	66321344, 66453056, 66584384, 66715328, 66846656, 66977728, 67108672,
	67239104, 67370432, 67501888, 67631296, 67763776, 67895104, 68026304,
	68157248, 68287936, 68419264, 68548288, 68681408, 68811968, 68942912,
	69074624, 69205568, 69337024, 69467584, 69599168, 69729472, 69861184,
	69989824, 70122944, 70253888, 70385344, 70515904, 70647232, 70778816,
	70907968, 71040832, 71171648, 71303104, 71432512, 71564992, 71695168,
	71826368, 71958464, 72089536, 72219712, 72350144, 72482624, 72613568,
	72744512, 72875584, 73006144, 73138112, 73268672, 73400128, 73530944,
	73662272, 73793344, 73924544, 74055104, 74185792, 74316992, 74448832,
	74579392, 74710976, 74841664, 74972864, 75102784, 75233344, 75364544,
	75497024, 75627584, 75759296, 75890624, 76021696, 76152256, 76283072,
	76414144, 76545856, 76676672, 76806976, 76937792, 77070016, 77200832,
	77331392, 77462464, 77593664, 77725376, 77856448, 77987776, 78118336,
	78249664, 78380992, 78511424, 78642496, 78773056, 78905152, 79033664,
	79166656, 79297472, 79429568, 79560512, 79690816, 79822784, 79953472,
	80084672, 80214208, 80346944, 80477632, 80608576, 80740288, 80870848,
	81002048, 81133504, 81264448, 81395648, 81525952, 81657536, 81786304,
	81919808, 82050112, 82181312, 82311616, 82443968, 82573376, 82705984,
	82835776, 82967744, 83096768, 83230528, 83359552, 83491264, 83622464,
	83753536, 83886016, 84015296, 84147776, 84277184, 84409792, 84540608,
	84672064, 84803008, 84934336, 85065152, 85193792, 85326784, 85458496,
	85589312, 85721024, 85851968, 85982656, 86112448, 86244416, 86370112,
	86506688, 86637632, 86769344, 86900672, 87031744, 87162304, 87293632,
	87424576, 87555392, 87687104, 87816896, 87947968, 88079168, 88211264,
	88341824, 88473152, 88603712, 88735424, 88862912, 88996672, 89128384,
	89259712, 89390272, 89521984, 89652544, 89783872, 89914816, 90045376,
	90177088, 90307904, 90438848, 90569152, 90700096, 90832832, 90963776,
	91093696, 91223744, 91356992, 91486784, 91618496, 91749824, 91880384,
	92012224, 92143552, 92273344, 92405696, 92536768, 92666432, 92798912,
	92926016, 93060544, 93192128, 93322816, 93453632, 93583936, 93715136,
	93845056, 93977792, 94109504, 94240448, 94371776, 94501184, 94632896,
	94764224, 94895552, 95023424, 95158208, 95287744, 95420224, 95550016,
	95681216, 95811904, 95943872, 96075328, 96203584, 96337856, 96468544,
	96599744, 96731072, 96860992, 96992576, 97124288, 97254848, 97385536,
	97517248, 97647808, 97779392, 97910464, 98041408, 98172608, 98303168,
	98434496, 98565568, 98696768, 98827328, 98958784, 99089728, 99220928,
	99352384, 99482816, 99614272, 99745472, 99876416, 100007104,
	100138048, 100267072, 100401088, 100529984, 100662592, 100791872,
	100925248, 101056064, 101187392, 101317952, 101449408, 101580608,
	101711296, 101841728, 101973824, 102104896, 102235712, 102366016,
	102498112, 102628672, 102760384, 102890432, 103021888, 103153472,
	103284032, 103415744, 103545152, 103677248, 103808576, 103939648,
	104070976, 104201792, 104332736, 104462528, 104594752, 104725952,
	104854592, 104988608, 105118912, 105247808, 105381184, 105511232,
	105643072, 105774784, 105903296, 106037056, 106167872, 106298944,
	106429504, 106561472, 106691392, 106822592, 106954304, 107085376,
	107216576, 107346368, 107478464, 107609792, 107739712, 107872192,
	108003136, 108131392, 108265408, 108396224, 108527168, 108657344,
	108789568, 108920384, 109049792, 109182272, 109312576, 109444928,
	109572928, 109706944, 109837888, 109969088, 110099648, 110230976,
	110362432, 110492992, 110624704, 110755264, 110886208, 111017408,
	111148864, 111279296, 111410752, 111541952, 111673024, 111803456,
	111933632, 112066496, 112196416, 112328512, 112457792, 112590784,
	112715968, 112852672, 112983616, 113114944, 113244224, 113376448,
	113505472, 113639104, 113770304, 113901376, 114031552, 114163264,
	114294592, 114425536, 114556864, 114687424, 114818624, 114948544,
	115080512, 115212224, 115343296, 115473472, 115605184, 115736128,
	115867072, 115997248, 116128576, 116260288, 116391488, 116522944,
	116652992, 116784704, 116915648, 117046208, 117178304, 117308608,
	117440192, 117569728, 117701824, 117833024, 117964096, 118094656,
	118225984, 118357312, 118489024, 118617536, 118749632, 118882112,
	119012416, 119144384, 119275328, 119406016, 119537344, 119668672,
	119798464, 119928896, 120061376, 120192832, 120321728, 120454336,
	120584512, 120716608, 120848192, 120979136, 121109056, 121241408,
	121372352, 121502912, 121634752, 121764416, 121895744, 122027072,
	122157632, 122289088, 122421184, 122550592, 122682944, 122813888,
	122945344, 123075776, 123207488, 123338048, 123468736, 123600704,
	123731264, 123861952, 123993664, 124124608, 124256192, 124386368,
	124518208, 124649024, 124778048, 124911296, 125041088, 125173696,
	125303744, 125432896, 125566912, 125696576, 125829056, 125958592,
	126090304, 126221248, 126352832, 126483776, 126615232, 126746432,
	126876608, 127008704, 127139392, 127270336, 127401152, 127532224,
	127663552, 127794752, 127925696, 128055232, 128188096, 128319424,
	128449856, 128581312, 128712256, 128843584, 128973632, 129103808,
	129236288, 129365696, 129498944, 129629888, 129760832, 129892288,
	130023104, 130154048, 130283968, 130416448, 130547008, 130678336,
	130807616, 130939456, 131071552, 131202112, 131331776, 131464384,
	131594048, 131727296, 131858368, 131987392, 132120256, 132250816,
	132382528, 132513728, 132644672, 132774976, 132905792, 133038016,
	133168832, 133299392, 133429312, 133562048, 133692992, 133823296,
	133954624, 134086336, 134217152, 134348608, 134479808, 134607296,
	134741056, 134872384, 135002944, 135134144, 135265472, 135396544,
	135527872, 135659072, 135787712, 135921472, 136052416, 136182848,
	136313792, 136444864, 136576448, 136707904, 136837952, 136970048,
	137099584, 137232064, 137363392, 137494208, 137625536, 137755712,
	137887424, 138018368, 138149824, 138280256, 138411584, 138539584,
	138672832, 138804928, 138936128, 139066688, 139196864, 139328704,
	139460032, 139590208, 139721024, 139852864, 139984576, 140115776,
	140245696, 140376512, 140508352, 140640064, 140769856, 140902336,
	141032768, 141162688, 141294016, 141426496, 141556544, 141687488,
	141819584, 141949888, 142080448, 142212544, 142342336, 142474432,
	142606144, 142736192, 142868288, 142997824, 143129408, 143258944,
	143392448, 143523136, 143653696, 143785024, 143916992, 144045632,
	144177856, 144309184, 144440768, 144570688, 144701888, 144832448,
	144965056, 145096384, 145227584, 145358656, 145489856, 145620928,
	145751488, 145883072, 146011456, 146144704, 146275264, 146407232,
	146538176, 146668736, 146800448, 146931392, 147062336, 147193664,
	147324224, 147455936, 147586624, 147717056, 147848768, 147979456,
	148110784, 148242368, 148373312, 148503232, 148635584, 148766144,
	148897088, 149028416, 149159488, 149290688, 149420224, 149551552,
	149683136, 149814976, 149943616, 150076352, 150208064, 150338624,
	150470464, 150600256, 150732224, 150862784, 150993088, 151125952,
	151254976, 151388096, 151519168, 151649728, 151778752, 151911104,
	152042944, 152174144, 152304704, 152435648, 152567488, 152698816,
	152828992, 152960576, 153091648, 153222976, 153353792, 153484096,
	153616192, 153747008, 153878336, 154008256, 154139968, 154270912,
	154402624, 154533824, 154663616, 154795712, 154926272, 155057984,
	155188928, 155319872, 155450816, 155580608, 155712064, 155843392,
	155971136, 156106688, 156237376, 156367424, 156499264, 156630976,
	156761536, 156892352, 157024064, 157155008, 157284416, 157415872,
	157545536, 157677248, 157810496, 157938112, 158071744, 158203328,
	158334656, 158464832, 158596288, 158727616, 158858048, 158988992,
	159121216, 159252416, 159381568, 159513152, 159645632, 159776192,
	159906496, 160038464, 160169536, 160300352, 160430656, 160563008,
	160693952, 160822208, 160956352, 161086784, 161217344, 161349184,
	161480512, 161611456, 161742272, 161873216, 162002752, 162135872,
	162266432, 162397888, 162529216, 162660032, 162790976, 162922048,
	163052096, 163184576, 163314752, 163446592, 163577408, 163707968,
	163839296, 163969984, 164100928, 164233024, 164364224, 164494912,
	164625856, 164756672, 164887616, 165019072, 165150016, 165280064,
	165412672, 165543104, 165674944, 165805888, 165936832, 166067648,
	166198336, 166330048, 166461248, 166591552, 166722496, 166854208,
	166985408, 167116736, 167246656, 167378368, 167508416, 167641024,
	167771584, 167903168, 168034112, 168164032, 168295744, 168427456,
	168557632, 168688448, 168819136, 168951616, 169082176, 169213504,
	169344832, 169475648, 169605952, 169738048, 169866304, 169999552,
	170131264, 170262464, 170393536, 170524352, 170655424, 170782016,
	170917696, 171048896, 171179072, 171310784, 171439936, 171573184,
	171702976, 171835072, 171966272, 172097216, 172228288, 172359232,
	172489664, 172621376, 172747712, 172883264, 173014208, 173144512,
	173275072, 173407424, 173539136, 173669696, 173800768, 173931712,
	174063424, 174193472, 174325696, 174455744, 174586816, 174718912,
	174849728, 174977728, 175109696, 175242688, 175374272, 175504832,
	175636288, 175765696, 175898432, 176028992, 176159936, 176291264,
	176422592, 176552512, 176684864, 176815424, 176946496, 177076544,
	177209152, 177340096, 177470528, 177600704, 177731648, 177864256,
	177994816, 178126528, 178257472, 178387648, 178518464, 178650176,
	178781888, 178912064, 179044288, 179174848, 179305024, 179436736,
	179568448, 179698496, 179830208, 179960512, 180092608, 180223808,
	180354752, 180485696, 180617152, 180748096, 180877504, 181009984,
	181139264, 181272512, 181402688, 181532608, 181663168, 181795136,
	181926592, 182057536, 182190016, 182320192, 182451904, 182582336,
	182713792, 182843072, 182976064, 183107264, 183237056, 183368384,
	183494848, 183631424, 183762752, 183893824, 184024768, 184154816,
	184286656, 184417984, 184548928, 184680128, 184810816, 184941248,
	185072704, 185203904, 185335616, 185465408, 185596352, 185727296,
	185859904, 185989696, 186121664, 186252992, 186383552, 186514112,
	186645952, 186777152, 186907328, 187037504, 187170112, 187301824,
	187429184, 187562048, 187693504, 187825472, 187957184, 188087104,
	188218304, 188349376, 188481344, 188609728, 188743616, 188874304,
	189005248, 189136448, 189265088, 189396544, 189528128, 189660992,
	189791936, 189923264, 190054208, 190182848, 190315072, 190447424,
	190577984, 190709312, 190840768, 190971328, 191102656, 191233472,
	191364032, 191495872, 191626816, 191758016, 191888192, 192020288,
	192148928, 192282176, 192413504, 192542528, 192674752, 192805952,
	192937792, 193068608, 193198912, 193330496, 193462208, 193592384,
	193723456, 193854272, 193985984, 194116672, 194247232, 194379712,
	194508352, 194641856, 194772544, 194900672, 195035072, 195166016,
	195296704, 195428032, 195558592, 195690304, 195818176, 195952576,
	196083392, 196214336, 196345792, 196476736, 196607552, 196739008,
	196869952, 197000768, 197130688, 197262784, 197394368, 197523904,
	197656384, 197787584, 197916608, 198049472, 198180544, 198310208,
	198442432, 198573632, 198705088, 198834368, 198967232, 199097792,
	199228352, 199360192, 199491392, 199621696, 199751744, 199883968,
	200014016, 200146624, 200276672, 200408128, 200540096, 200671168,
	200801984, 200933312, 201062464, 201194944, 201326144, 201457472,
	201588544, 201719744, 201850816, 201981632, 202111552, 202244032,
	202374464, 202505152, 202636352, 202767808, 202898368, 203030336,
	203159872, 203292608, 203423296, 203553472, 203685824, 203816896,
	203947712, 204078272, 204208192, 204341056, 204472256, 204603328,
	204733888, 204864448, 204996544, 205125568, 205258304, 205388864,
	205517632, 205650112, 205782208, 205913536, 206044736, 206176192,
	206307008, 206434496, 206569024, 206700224, 206831168, 206961856,
	207093056, 207223616, 207355328, 207486784, 207616832, 207749056,
	207879104, 208010048, 208141888, 208273216, 208404032, 208534336,
	208666048, 208796864, 208927424, 209059264, 209189824, 209321792,
	209451584, 209582656, 209715136, 209845568, 209976896, 210106432,
	210239296, 210370112, 210501568, 210630976, 210763712, 210894272,
	211024832, 211156672, 211287616, 211418176, 211549376, 211679296,
	211812032, 211942592, 212074432, 212204864, 212334016, 212467648,
	212597824, 212727616, 212860352, 212991424, 213120832, 213253952,
	213385024, 213515584, 213645632, 213777728, 213909184, 214040128,
	214170688, 214302656, 214433728, 214564544, 214695232, 214826048,
	214956992, 215089088, 215219776, 215350592, 215482304, 215613248,
	215743552, 215874752, 216005312, 216137024, 216267328, 216399296,
	216530752, 216661696, 216790592, 216923968, 217054528, 217183168,
	217316672, 217448128, 217579072, 217709504, 217838912, 217972672,
	218102848, 218233024, 218364736, 218496832, 218627776, 218759104,
	218888896, 219021248, 219151936, 219281728, 219413056, 219545024,
	219675968, 219807296, 219938624, 220069312, 220200128, 220331456,
	220461632, 220592704, 220725184, 220855744, 220987072, 221117888,
	221249216, 221378368, 221510336, 221642048, 221772736, 221904832,
	222031808, 222166976, 222297536, 222428992, 222559936, 222690368,
	222820672, 222953152, 223083968, 223213376, 223345984, 223476928,
	223608512, 223738688, 223869376, 224001472, 224132672, 224262848,
	224394944, 224524864, 224657344, 224788288, 224919488, 225050432,
	225181504, 225312704, 225443776, 225574592, 225704768, 225834176,
	225966784, 226097216, 226229824, 226360384, 226491712, 226623424,
	226754368, 226885312, 227015104, 227147456, 227278528, 227409472,
	227539904, 227669696, 227802944, 227932352, 228065216, 228196288,
	228326464, 228457792, 228588736, 228720064, 228850112, 228981056,
	229113152, 229243328, 229375936, 229505344, 229636928, 229769152,
	229894976, 230030272, 230162368, 230292416, 230424512, 230553152,
	230684864, 230816704, 230948416, 231079616, 231210944, 231342016,
	231472448, 231603776, 231733952, 231866176, 231996736, 232127296,
	232259392, 232388672, 232521664, 232652608, 232782272, 232914496,
	233043904, 233175616, 233306816, 233438528, 233569984, 233699776,
	233830592, 233962688, 234092224, 234221888, 234353984, 234485312,
	234618304, 234749888, 234880832, 235011776, 235142464, 235274048,
	235403456, 235535936, 235667392, 235797568, 235928768, 236057152,
	236190272, 236322752, 236453312, 236583616, 236715712, 236846528,
	236976448, 237108544, 237239104, 237371072, 237501632, 237630784,
	237764416, 237895232, 238026688, 238157632, 238286912, 238419392,
	238548032, 238681024, 238812608, 238941632, 239075008, 239206336,
	239335232, 239466944, 239599168, 239730496, 239861312, 239992384,
	240122816, 240254656, 240385856, 240516928, 240647872, 240779072,
	240909632, 241040704, 241171904, 241302848, 241433408, 241565248,
	241696192, 241825984, 241958848, 242088256, 242220224, 242352064,
	242481856, 242611648, 242744896, 242876224, 243005632, 243138496,
	243268672, 243400384, 243531712, 243662656, 243793856, 243924544,
	244054592, 244187072, 244316608, 244448704, 244580032, 244710976,
	244841536, 244972864, 245104448, 245233984, 245365312, 245497792,
	245628736, 245759936, 245889856, 246021056, 246152512, 246284224,
	246415168, 246545344, 246675904, 246808384, 246939584, 247070144,
	247199552, 247331648, 247463872, 247593536, 247726016, 247857088,
	247987648, 248116928, 248249536, 248380736, 248512064, 248643008,
	248773312, 248901056, 249036608, 249167552, 249298624, 249429184,
	249560512, 249692096, 249822784, 249954112, 250085312, 250215488,
	250345792, 250478528, 250608704, 250739264, 250870976, 251002816,
	251133632, 251263552, 251395136, 251523904, 251657792, 251789248,
	251919424, 252051392, 252182464, 252313408, 252444224, 252575552,
	252706624, 252836032, 252968512, 253099712, 253227584, 253361728,
	253493056, 253623488, 253754432, 253885504, 254017216, 254148032,
	254279488, 254410432, 254541376, 254672576, 254803264, 254933824,
	255065792, 255196736, 255326528, 255458752, 255589952, 255721408,
	255851072, 255983296, 256114624, 256244416, 256374208, 256507712,
	256636096, 256768832, 256900544, 257031616, 257162176, 257294272,
	257424448, 257555776, 257686976, 257818432, 257949632, 258079552,
	258211136, 258342464, 258473408, 258603712, 258734656, 258867008,
	258996544, 259127744, 259260224, 259391296, 259522112, 259651904,
	259784384, 259915328, 260045888, 260175424, 260308544, 260438336,
	260570944, 260700992, 260832448, 260963776, 261092672, 261226304,
	261356864, 261487936, 261619648, 261750592, 261879872, 262011968,
	262143424, 262274752, 262404416, 262537024, 262667968, 262799296,
	262928704, 263061184, 263191744, 263322944, 263454656, 263585216,
	263716672, 263847872, 263978944, 264108608, 264241088, 264371648,
	264501184, 264632768, 264764096, 264895936, 265024576, 265158464,
	265287488, 265418432, 265550528, 265681216, 265813312, 265943488,
	266075968, 266206144, 266337728, 266468032, 266600384, 266731072,
	266862272, 266993344, 267124288, 267255616, 267386432, 267516992,
	267648704, 267777728, 267910592, 268040512, 268172096, 268302784,
	268435264, 268566208, 268696256, 268828096, 268959296, 269090368,
	269221312, 269352256, 269482688, 269614784, 269745856, 269876416,
	270007616, 270139328, 270270272, 270401216, 270531904, 270663616,
	270791744, 270924736, 271056832, 271186112, 271317184, 271449536,
	271580992, 271711936, 271843136, 271973056, 272105408, 272236352,
	272367296, 272498368, 272629568, 272759488, 272891456, 273022784,
	273153856, 273284672, 273415616, 273547072, 273677632, 273808448,
	273937088, 274071488, 274200896, 274332992, 274463296, 274595392,
	274726208, 274857536, 274988992, 275118656, 275250496, 275382208,
	275513024, 275643968, 275775296, 275906368, 276037184, 276167872,
	276297664, 276429376, 276560576, 276692672, 276822976, 276955072,
	277085632, 277216832, 277347008, 277478848, 277609664, 277740992,
	277868608, 278002624, 278134336, 278265536, 278395328, 278526784,
	278657728, 278789824, 278921152, 279052096, 279182912, 279313088,
	279443776, 279576256, 279706048, 279838528, 279969728, 280099648,
	280230976, 280361408, 280493632, 280622528, 280755392, 280887104,
	281018176, 281147968, 281278912, 281411392, 281542592, 281673152,
	281803712, 281935552, 282066496, 282197312, 282329024, 282458816,
	282590272, 282720832, 282853184, 282983744, 283115072, 283246144,
	283377344, 283508416, 283639744, 283770304, 283901504, 284032576,
	284163136, 284294848, 284426176, 284556992, 284687296, 284819264,
	284950208, 285081536}
