package servedir

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/andybalholm/brotli"
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

	compression, err := cmd.Flags().GetBool("compression")
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Serving directory %s on port %s\n", dir, port)

	handler := http.FileServer(http.Dir(dir))

	if compression {
		compressionLevel, err := cmd.Flags().GetInt("compression-level")
		if err != nil {
			log.Fatalln(err)
		}

		if compressionLevel < brotli.BestSpeed {
			compressionLevel = brotli.BestSpeed
		}

		if compressionLevel > brotli.BestCompression {
			compressionLevel = brotli.BestCompression
		}

		handler = MustNewBrotliLevelHandler(compressionLevel)(handler)
	}

	err = http.ListenAndServe(port, handler)
	log.Fatalln(err)
}
