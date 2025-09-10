package config

// SelectApp selects an app from the configuration based on the provided key.
// Selection priority:
//  1. If key is provided and matches an app ID, returns that app
//  2. If no key match, returns the app marked as default
//  3. If no default app, returns the first app in the list
//  4. Returns nil if config is nil or has no apps
func (u *User) SelectApp(key string) *App {
	if u == nil || len(u.Apps) == 0 {
		return nil
	}
	var app *App
	for _, a := range u.Apps {
		if key != "" && a.Name == key {
			return a
		}
		if app == nil && a.Default {
			app = a
		}
	}
	if app != nil {
		return app
	}
	return u.Apps[0]
}

func (c *Config) SelectUser(key string) *User {
	if c == nil || len(c.Users) == 0 {
		return nil
	}
	var user *User
	for _, a := range c.Users {
		if key != "" && a.Login == key {
			return a
		}
		if user == nil && a.Default {
			user = a
		}
	}
	if user != nil {
		return user
	}
	return c.Users[0]
}
