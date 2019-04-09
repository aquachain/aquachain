:: Copyright 2019 The Aquachain Authors. All rights reserved.
:: Use of this source code is governed by a BSD-style
:: license that can be found in the LICENSE file.

@echo off

set GOBUILDFAIL=0
if exist make.bat goto ok
echo Must run make.bat from Aquachain source directory.
goto fail

:ok
set CGO_ENABLED=0
set GO111MODULE=on
echo Building Aquachain (aquachain.exe in Desktop)
go.exe build -v -mod vendor -o "%USERPROFILE%\Desktop\aquachain.exe" .\cmd\aquachain
if errorlevel 1 goto fail
if errorlevel 1 goto fail
goto end

:fail
set GOBUILDFAIL=1
if x%GOBUILDEXIT%==x1 exit %GOBUILDFAIL%
:end
echo "Successfully Built 'aquachain.exe"
