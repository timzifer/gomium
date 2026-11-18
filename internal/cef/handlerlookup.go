package cef

// clientWhere returns the first registered client for which match is true,
// or nil. Used by the per-handler callbacks to map a CEF handler pointer back
// to its client.
func clientWhere(match func(c *client) bool) *client {
	app := getInstance()
	if app == nil {
		return nil
	}

	app.clientMutex.Lock()
	defer app.clientMutex.Unlock()

	for _, c := range app.clients {
		c.Lock()
		ok := match(c)
		c.Unlock()
		if ok {
			return c
		}
	}
	return nil
}
