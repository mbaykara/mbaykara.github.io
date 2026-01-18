.PHONY: markdownlint markdownlint-fix help

help:
	@echo "Available targets:"
	@echo "  markdownlint      - Check markdown files for formatting issues"
	@echo "  markdownlint-fix  - Fix markdown formatting issues automatically"

markdownlint:
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 "**/*.md" "!.github/**/*.md" "!node_modules/**/*.md" --config .markdownlint.json; \
	else \
		echo "markdownlint-cli2 not found. Install it with: npm install -g markdownlint-cli2"; \
		exit 1; \
	fi

markdownlint-fix:
	@if command -v markdownlint-cli2 >/dev/null 2>&1; then \
		markdownlint-cli2 --fix "**/*.md" "!.github/**/*.md" "!node_modules/**/*.md" --config .markdownlint.json; \
	else \
		echo "markdownlint-cli2 not found. Install it with: npm install -g markdownlint-cli2"; \
		exit 1; \
	fi
