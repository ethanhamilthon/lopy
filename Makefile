.PHONY: install dev clean

BINDIR ?= $(HOME)/.local/bin
DEV_HOME ?= $(HOME)/.lopydev

# Build lopy into BINDIR (default ~/.local/bin) so it can be run from anywhere.
install:
	mkdir -p $(BINDIR)
	go build -o $(BINDIR)/lopy .
	@echo "installed $(BINDIR)/lopy"

# Run from source with an isolated dev home (settings, prompts, sessions).
dev:
	LOPY_HOME=$(DEV_HOME) go run . $(ARGS)

# Remove the dev home.
clean:
	rm -rf $(DEV_HOME)
