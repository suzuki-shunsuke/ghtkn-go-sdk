package config

// SelectApp selects an app from the configuration based on the provided key.
// Selection priority:
//  1. If owner is provided and matches an app's GitOwner, returns that app
//  2. If key is provided and matches an app name, returns that app
//  3. If key is provided but does not match, returns nil
//  4. If both key and owner are empty, returns the first app in the list
//  5. Returns nil if config is nil or has no apps
func SelectApp(cfg *Config, key, owner string) *App {
	if cfg == nil || len(cfg.Apps) == 0 {
		return nil
	}
	if owner != "" {
		for _, a := range cfg.Apps {
			if a.GitOwner == owner {
				return a
			}
		}
	}
	if key == "" {
		return cfg.Apps[0]
	}
	for _, a := range cfg.Apps {
		if a.Name == key {
			return a
		}
	}
	return nil
}
