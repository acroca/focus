package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"
	"github.com/samber/lo"
	"golang.design/x/hotkey"
	"golang.design/x/hotkey/mainthread"
)

type Config struct {
	Bindings []Binding `toml:"bindings"`
}

type Binding struct {
	Key     string   `toml:"key"`
	App     string   `toml:"app"`
	Modifer []string `toml:"modifiers"`
}

// Add these constants and maps near the top of the file
const (
	keyMin = hotkey.Key0
	keyMax = hotkey.KeyF20
)

var keyMap = map[string]hotkey.Key{
	// Numbers
	"0": hotkey.Key0,
	"1": hotkey.Key1,
	"2": hotkey.Key2,
	"3": hotkey.Key3,
	"4": hotkey.Key4,
	"5": hotkey.Key5,
	"6": hotkey.Key6,
	"7": hotkey.Key7,
	"8": hotkey.Key8,
	"9": hotkey.Key9,

	// Letters
	"a": hotkey.KeyA,
	"b": hotkey.KeyB,
	"c": hotkey.KeyC,
	"d": hotkey.KeyD,
	"e": hotkey.KeyE,
	"f": hotkey.KeyF,
	"g": hotkey.KeyG,
	"h": hotkey.KeyH,
	"i": hotkey.KeyI,
	"j": hotkey.KeyJ,
	"k": hotkey.KeyK,
	"l": hotkey.KeyL,
	"m": hotkey.KeyM,
	"n": hotkey.KeyN,
	"o": hotkey.KeyO,
	"p": hotkey.KeyP,
	"q": hotkey.KeyQ,
	"r": hotkey.KeyR,
	"s": hotkey.KeyS,
	"t": hotkey.KeyT,
	"u": hotkey.KeyU,
	"v": hotkey.KeyV,
	"w": hotkey.KeyW,
	"x": hotkey.KeyX,
	"y": hotkey.KeyY,
	"z": hotkey.KeyZ,

	// Function keys
	"f1":  hotkey.KeyF1,
	"f2":  hotkey.KeyF2,
	"f3":  hotkey.KeyF3,
	"f4":  hotkey.KeyF4,
	"f5":  hotkey.KeyF5,
	"f6":  hotkey.KeyF6,
	"f7":  hotkey.KeyF7,
	"f8":  hotkey.KeyF8,
	"f9":  hotkey.KeyF9,
	"f10": hotkey.KeyF10,
	"f11": hotkey.KeyF11,
	"f12": hotkey.KeyF12,

	// Special keys
	"tab":    hotkey.KeyTab,
	"space":  hotkey.KeySpace,
	"return": hotkey.KeyReturn,
}
var keyKeys = lo.Keys(keyMap)

var modMap = map[string]hotkey.Modifier{
	"cmd":     hotkey.ModCmd,
	"command": hotkey.ModCmd,
	"ctrl":    hotkey.ModCtrl,
	"control": hotkey.ModCtrl,
	"alt":     hotkey.ModOption,
	"option":  hotkey.ModOption,
	"shift":   hotkey.ModShift,
}

var modKeys = lo.Keys(modMap)

// loadConfig loads the configuration from ~/.config/focus/config.toml
func loadConfig() (*Config, error) {
	configDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, ".config", "focus", "config.toml")

	// Try to load config file
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Printf("Config file not found at %s. Using default configuration.\n", configPath)
		fmt.Println("To customize bindings, create a config file with the following format:")
		fmt.Println("[[bindings]]")
		fmt.Println("app = \"Cursor\"")
		fmt.Println("key = \"1\"")
		fmt.Println("modifiers = [\"cmd\"]")
		os.Exit(1)
	}

	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	mainthread.Init(fn)
}

type hk struct {
	hk  *hotkey.Hotkey
	app string
}

func (h *hk) register() {
	if err := h.hk.Register(); err != nil {
		panic(err)
	}
}

func (h *hk) handle() {
	for range h.hk.Keydown() {
		open(h.app)
	}
}

func (h *hk) unregister() {
	if err := h.hk.Unregister(); err != nil {
		panic(err)
	}
}

func createHotkeys(config *Config) ([]*hk, error) {
	var keys []*hk

	for _, binding := range config.Bindings {
		// Convert string key to hotkey.Key
		k, ok := keyMap[strings.ToLower(binding.Key)]
		if !ok {
			log.Printf("Error: unsupported key %q\nValid keys are: %v", binding.Key, keyKeys)
			os.Exit(1)
		}

		// Convert string modifier to hotkey.Modifier
		var mods []hotkey.Modifier
		for _, mod := range binding.Modifer {
			mod, ok := modMap[strings.ToLower(mod)]
			if !ok {
				log.Printf("Error: unsupported modifier %q\nValid modifiers are: %v", mod, modKeys)
				os.Exit(1)
			}
			mods = append(mods, mod)
		}

		keys = append(keys, &hk{
			hk:  hotkey.New(mods, k),
			app: binding.App,
		})
	}

	return keys, nil
}

func fn() {
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	keys, err := createHotkeys(config)
	if err != nil {
		log.Fatalf("Failed to create hotkeys: %v", err)
	}

	log.Println("registering hotkeys")
	for _, k := range keys {
		k.register()
		go k.handle()
	}
	log.Println("hotkeys registered")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("unregistering hotkeys")
	for _, k := range keys {
		k.unregister()
	}
	log.Println("hotkeys unregistered")
}

func open(name string) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	objc.WithAutoreleasePool(func() {
		ws := appkit.Workspace_SharedWorkspace()
		for _, app := range ws.RunningApplications() {
			if app.LocalizedName() == name {
				app.ActivateWithOptions(appkit.ApplicationActivateIgnoringOtherApps)
				return
			}
		}
	})
}
