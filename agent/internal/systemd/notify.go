package systemd

import (
	"net"
	"os"
	"strings"
)

// Notify sends a notification message to systemd when NOTIFY_SOCKET is present.
// It is a no-op when running outside systemd.
func Notify(state string) error {
	socket := os.Getenv("NOTIFY_SOCKET")
	if socket == "" {
		return nil
	}

	addr := socket
	if strings.HasPrefix(socket, "@") {
		addr = "\x00" + socket[1:]
	}

	conn, err := net.DialUnix("unixgram", nil, &net.UnixAddr{Name: addr, Net: "unixgram"})
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(state))
	return err
}
