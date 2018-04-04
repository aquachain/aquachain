package main

import (
	"fmt"
	"os"

	"github.com/therecipe/qt/widgets"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	app = cli.NewApp()
)

func init() {
	app.Usage = ""
	app.Name = "aquachain-qt"
	app.Description = "Aquachain GUI Wallet (QT)"
	app.Action = launchWindow
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println("fatal", err)
		os.Exit(111)
	}
}

type Account struct {
}

func (a *Account) SendTransaction(tx string) (hash string, err error) {
	return "0x000000000000000000000000000000000000000000", nil
}

func launchWindow(ctx *cli.Context) error {
	// Create application
	//client := aquaclient.
	guiapp := widgets.NewQApplication(len(os.Args), os.Args)
	//acct := &Account{}
	// Create main window
	window := widgets.NewQMainWindow(nil, 0)
	window.SetWindowTitle("Aquachain G-Wallet")
	window.SetMinimumSize2(200, 200)

	// Create main layout
	layout := widgets.NewQVBoxLayout()

	// Create main widget and set the layout
	mainWidget := widgets.NewQWidget(nil, 0)
	mainWidget.SetLayout(layout)

	// Create a line edit and add it to the layout

	input := widgets.NewQLineEdit(nil)
	input.SetPlaceholderText("Send AQUA where?")
	layout.AddWidget(input, 0, 0)
	inputAmount := widgets.NewQLineEdit(nil)
	inputAmount.SetPlaceholderText("How much to send?")
	layout.AddWidget(inputAmount, 0, 0)

	// Create a button and add it to the layout
	button := widgets.NewQPushButton2("Sign Transaction!", nil)
	layout.AddWidget(button, 0, 0)

	// Connect event for button
	button.ConnectClicked(func(checked bool) {
		msg := "not sent"
		hash, err := acct.SendTransaction("{from: aqua.coinbase, to: '" + input.Text() + "', value: web3.toWei(" + inputAmount.Text() + ", 'aqua'}")
		if err != nil {
			msg = "error: " + err.Error()
		} else {
			msg = "sent: " + hash
		}
		widgets.QMessageBox_Information(nil, "OK", msg,
			widgets.QMessageBox__Ok, widgets.QMessageBox__Ok)
	})

	// Set main widget as the central widget of the window
	window.SetCentralWidget(mainWidget)

	// Show the window
	window.Show()

	// Execute app
	guiapp.Exec()
}
