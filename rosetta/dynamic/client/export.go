package client

import "io"

// Export exports the *Client to the target io.Writer
func Export(w io.Writer, c *Client) error {
	panic("")
}

// Import imports the *Client from an io.Reader
func Import(r io.Reader) (c *Client, err error) {
	panic("")
}

type exportedInfoProvider struct {
}
