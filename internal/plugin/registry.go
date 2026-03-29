package plugin

import "sync"

var (
	mu      sync.RWMutex
	plugins = map[string]Plugin{}
)

// Register adds a plugin to the global registry.
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	plugins[p.Name()] = p
}

// Get returns a plugin by ecosystem name.
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := plugins[name]
	return p, ok
}

// All returns all registered plugins.
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	result := make([]Plugin, 0, len(plugins))
	for _, p := range plugins {
		result = append(result, p)
	}
	return result
}

// DetectAll returns plugins that detect their ecosystem in the given project root.
func DetectAll(projectRoot string) []Plugin {
	mu.RLock()
	defer mu.RUnlock()
	var detected []Plugin
	for _, p := range plugins {
		if ok, _ := p.Detect(projectRoot); ok {
			detected = append(detected, p)
		}
	}
	return detected
}
