package config

// ResolveApp resolves the app ghtkn should use from cfg by applying the selection
// priority:
//  1. If owner is non-empty and matches an app's GitOwner, that app.
//  2. If key is empty (regardless of whether owner matched), the first app in the list
//     (the default app).
//  3. Otherwise the app whose Name equals key (nil when none matches).
//
// It returns nil when cfg is nil or has no apps. It is exported so callers (e.g. the
// ghtkn CLI's `info` command) resolve the app exactly as token retrieval does, instead
// of reimplementing this logic.
func ResolveApp(cfg *Config, key, owner string) *App {
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
