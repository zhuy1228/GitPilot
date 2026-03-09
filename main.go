package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/zhuy1228/GitPilot/internal/app"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	log.Println("GitPilot 启动中...")

	appService := app.New()

	wailsApp := application.New(application.Options{
		Name:        "GitPilot",
		Description: "Git Repository Management Tool",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "GitPilot",
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            1280,
		Height:           900,
	})

	appService.SetApplication(wailsApp)

	// 窗口关闭时隐藏到托盘而不是退出
	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		log.Println("窗口关闭，隐藏到系统托盘")
		mainWindow.Hide()
		event.Cancel()
	})

	// 创建系统托盘
	systray := wailsApp.SystemTray.New()

	// 创建托盘菜单
	trayMenu := wailsApp.NewMenu()
	trayMenu.Add("显示主窗口").OnClick(func(ctx *application.Context) {
		log.Println("托盘菜单：显示主窗口")
		mainWindow.Show()
		mainWindow.Focus()
	})
	trayMenu.AddSeparator()
	trayMenu.Add("退出").OnClick(func(ctx *application.Context) {
		log.Println("托盘菜单：退出应用")
		wailsApp.Quit()
	})

	systray.SetMenu(trayMenu)

	// 单击托盘图标显示主窗口
	systray.OnClick(func() {
		log.Println("托盘图标被点击")
		mainWindow.Show()
		mainWindow.Focus()
	})

	log.Println("系统托盘创建成功")

	go handleShutdownSignals()

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}

func handleShutdownSignals() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	log.Printf("接收到信号: %v, 正在退出...", sig)
	os.Exit(0)
}
