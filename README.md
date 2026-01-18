
# Personal Blog

A minimalist Markdown-based blog built with Go.

## Features

- Markdown-based posts with syntax highlighting
- Static site generation for GitHub Pages
- Dark terminal-inspired theme

## Usage

### Development Server

```bash
go run main.go
```

Server runs on `http://localhost:8090`

### Generate Static Site

```bash
go run main.go --generate
```

Generates static HTML files in the `public/` directory.

## Project Structure

- `posts/` - Markdown blog posts
- `nav/` - Navigation pages (about, contact)
- `templates/` - Go HTML templates
- `public/` - Generated static site (after running --generate)

## Adding Posts

Add Markdown files to the `posts/` directory. Posts are automatically discovered and sorted by modification date.

## Markdown Format Checking

This project includes markdown format checking to ensure consistent formatting across all markdown files.

### Installation

Install `markdownlint-cli2` globally:

```bash
npm install -g markdownlint-cli2
```

### Usage

```bash
# Check markdown files for formatting issues
make markdownlint

# Automatically fix markdown formatting issues
make markdownlint-fix
```

### CI/CD

Markdown linting runs automatically on all pushes and pull requests via GitHub Actions. The workflow checks all `.md` files in the repository using the configuration in `.markdownlint.json`.
