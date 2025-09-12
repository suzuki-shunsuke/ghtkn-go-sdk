package config

// SelectApp selects an app from the configuration based on the provided key.
// Selection priority:
//  1. If key is provided and matches an app name, returns that app
//  2. If no key match or key is empty, returns the first app in the list
//  3. Returns nil if config is nil or has no apps
func (c *Config) SelectApp(key string) *App {
	if c == nil || len(c.Apps) == 0 {
		return nil
	}
	for _, a := range c.Apps {
		if key != "" && a.Name == key {
			return a
		}
	}
	return c.Apps[0]
}
