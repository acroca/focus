.PHONY: build
build:
	go build -o focus main.go

.PHONY: reload
reload:
	cp com.example.focus.plist ~/Library/LaunchAgents/
	launchctl unload ~/Library/LaunchAgents/com.example.focus.plist
	launchctl load ~/Library/LaunchAgents/com.example.focus.plist
	launchctl start ~/Library/LaunchAgents/com.example.focus.plist
