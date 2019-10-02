package servedir

import (
	"fmt"
	"log"
	"net/http"

	"github.com/spf13/cobra"
)

func Serve(cmd *cobra.Command, args []string) {
	log.Println("Opening File Server")
	portInt, err := cmd.Flags().GetInt("port")
	if err != nil {
		log.Fatalln(err)
	}
	portStr := fmt.Sprintf(":%v", portInt)

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Serving directory %s on port %s\n", dir, portStr)

	err = http.ListenAndServe(portStr, http.FileServer(http.Dir(dir)))
	log.Fatalln(err)
}
