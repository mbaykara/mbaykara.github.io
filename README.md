
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