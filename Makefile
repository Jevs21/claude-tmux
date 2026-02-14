BINARY := claude-tmux
CMD    := ./cmd/claude-tmux
HOOK   := $(shell pwd)/hooks/claude-tmux-hook.sh

.PHONY: build test run clean install-hook

build:
	go build -o $(BINARY) $(CMD)

test:
	go test -v ./...

run: build
	./$(BINARY)

clean:
	rm -f $(BINARY)

install-hook:
	@echo "Hook script: $(HOOK)"
	@echo ""
	@echo "Add the following to ~/.claude/settings.json:"
	@echo ""
	@echo '{'
	@echo '  "hooks": {'
	@echo '    "SessionStart": [{ "hooks": [{ "type": "command", "command": "$(HOOK) session-start" }] }],'
	@echo '    "SessionEnd": [{ "hooks": [{ "type": "command", "command": "$(HOOK) session-end" }] }],'
	@echo '    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "$(HOOK) user-prompt-submit" }] }],'
	@echo '    "Stop": [{ "hooks": [{ "type": "command", "command": "$(HOOK) stop" }] }],'
	@echo '    "PreToolUse": [{ "hooks": [{ "type": "command", "command": "$(HOOK) pre-tool-use" }] }],'
	@echo '    "PostToolUse": [{ "hooks": [{ "type": "command", "command": "$(HOOK) post-tool-use" }] }],'
	@echo '    "PostToolUseFailure": [{ "hooks": [{ "type": "command", "command": "$(HOOK) post-tool-use-failure" }] }],'
	@echo '    "PermissionRequest": [{ "hooks": [{ "type": "command", "command": "$(HOOK) permission-request" }] }],'
	@echo '    "Notification": ['
	@echo '      { "matcher": "idle_prompt", "hooks": [{ "type": "command", "command": "$(HOOK) notification-idle" }] },'
	@echo '      { "matcher": "permission_prompt", "hooks": [{ "type": "command", "command": "$(HOOK) notification-permission" }] },'
	@echo '      { "matcher": "elicitation_dialog", "hooks": [{ "type": "command", "command": "$(HOOK) notification-elicitation" }] }'
	@echo '    ]'
	@echo '  }'
	@echo '}'
