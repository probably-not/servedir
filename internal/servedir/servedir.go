package servedir

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"
)

func Serve(cmd *cobra.Command, args []string) {
	log.Println("Opening File Server")
	portFlag, err := cmd.Flags().GetInt("port")
	if err != nil {
		log.Fatalln(err)
	}

	port := net.JoinHostPort("", strconv.Itoa(portFlag))

	dir, err := cmd.Flags().GetString("dir")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Serving directory %s on port %s\n", dir, port)

	err = http.ListenAndServe(port, http.FileServer(http.Dir(dir)))
	log.Fatalln(err)
}
