/*
Copyright © 2026 abdul hamid <abdulachik@icloud.com>
*/
package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"time"

	"github.com/abdul-hamid-achik/noted/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web interface",
	Long:  "Start a local web server with a browser-based note editor and monitoring dashboard.",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetInt("port")
		open, _ := cmd.Flags().GetBool("open")

		addr := fmt.Sprintf(":%d", port)
		srv := web.NewServer(database, conn)

		if open {
			go func() {
				time.Sleep(500 * time.Millisecond)
				openBrowser(fmt.Sprintf("http://localhost:%d", port))
			}()
		}

		fmt.Printf("noted web interface: http://localhost:%d\n", port)
		return srv.Run(cmd.Context(), addr)
	},
}

func init() {
	serveCmd.Flags().IntP("port", "p", 3000, "Port to listen on")
	serveCmd.Flags().Bool("open", false, "Open browser automatically")
	rootCmd.AddCommand(serveCmd)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}
	_ = cmd.Start()
}
