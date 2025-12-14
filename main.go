package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// PostFrontmatter represents the frontmatter data in markdown files.
type PostFrontmatter struct {
	Title string    `yaml:"title"`
	Date  time.Time `yaml:"date"`
}

// Post represents a blog post with a title, date, and content.
type PostData struct {
	Title   string
	Date    time.Time
	Slug    string
	Content template.HTML // Content after converting from Markdown
}

// TemplateData holds the data passed to the template.
type TemplateData struct {
	Title string
	Posts []PostData
}

// LoadPostFromFile loads a post from a file, parsing frontmatter for title and date.
func LoadPostFromFile(filePath string) (PostData, error) {
	mdBytes, err := os.ReadFile(filePath)
	if err != nil {
		return PostData{}, err
	}

	// Parse frontmatter
	var fm PostFrontmatter
	remainingMd, err := frontmatter.Parse(strings.NewReader(string(mdBytes)), &fm)
	if err != nil {
		// If frontmatter parsing fails, continue with empty frontmatter
		remainingMd = mdBytes
		fm = PostFrontmatter{}
	}

	// Get file info for fallback date
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return PostData{}, err
	}

	// Extract slug from filename
	filename := filepath.Base(filePath)
	slug := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Determine title: use frontmatter title, or cleaned filename
	title := fm.Title
	if title == "" {
		title = CleanTitle(filename)
	}

	// Determine date: use frontmatter date, or file modification time
	date := fm.Date
	if date.IsZero() {
		date = fileInfo.ModTime()
	}

	// Convert markdown to HTML
	markdown := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
		),
	)
	var buf bytes.Buffer
	err = markdown.Convert([]byte(remainingMd), &buf)
	if err != nil {
		return PostData{}, err
	}

	return PostData{
		Title:   title,
		Date:    date,
		Slug:    slug,
		Content: template.HTML(buf.String()),
	}, nil
}

// RenderMarkdown converts Markdown content to HTML (kept for backward compatibility).
func RenderMarkdown(filePath string) (template.HTML, error) {
	mdBytes, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	markdown := goldmark.New(
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
			),
		),
	)
	var fm PostFrontmatter
	remainingMd, err := frontmatter.Parse(strings.NewReader(string(mdBytes)), &fm)
	if err != nil {
		remainingMd = mdBytes
	}

	var buf bytes.Buffer
	err = markdown.Convert(remainingMd, &buf)
	if err != nil {
		panic(err)
	}
	return template.HTML(buf.String()), nil
}
func CleanTitle(filename string) string {
	// Remove the extension (.md) if present
	title := strings.TrimSuffix(filename, filepath.Ext(filename))

	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")

	// Capitalize the first letter of each word
	title = cases.Title(language.English).String(title)

	return title
}

func loadPostsFromDirectory(pattern string) ([]PostData, error) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var posts []PostData
	for _, file := range files {
		post, err := LoadPostFromFile(file)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	// Sort posts by date (latest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	return posts, nil
}

// LoadBlogPosts loads the blog posts from Markdown files and sorts them by date.
func LoadBlogPosts() ([]PostData, error) {
	return loadPostsFromDirectory("posts/*.md")
}

// LoadThoughtsPosts loads thoughts blog posts from Markdown files and sorts them by date.
func LoadThoughtsPosts() ([]PostData, error) {
	return loadPostsFromDirectory("thoughts/*.md")
}

func main() {
	// Check if we should generate static files instead of running a server
	if len(os.Args) > 1 && os.Args[1] == "--generate" {
		if err := GenerateStaticSite("public"); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Static site generated successfully!")
		return
	}

	http.HandleFunc("/", HomeHandler)
	http.HandleFunc("/about", AboutHandler)
	http.HandleFunc("/thoughts", ThoughtsHandler)
	http.HandleFunc("/post/", PostHandler)
	fmt.Println("Server is running...")
	log.Fatal(http.ListenAndServe(":8090", nil))
}

// GenerateStaticSite generates static HTML files for all pages
func GenerateStaticSite(outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Generate index.html (home page)
	if err := generatePage(outputDir, "index.html", func(w http.ResponseWriter) error {
		HomeHandler(w, &http.Request{})
		return nil
	}); err != nil {
		return err
	}

	// Generate about.html
	if err := generatePage(outputDir, "about.html", func(w http.ResponseWriter) error {
		AboutHandler(w, &http.Request{})
		return nil
	}); err != nil {
		return err
	}

	// Generate thoughts.html
	if err := generatePage(outputDir, "thoughts.html", func(w http.ResponseWriter) error {
		ThoughtsHandler(w, &http.Request{})
		return nil
	}); err != nil {
		return err
	}
	// Generate post pages from posts directory
	files, err := filepath.Glob("posts/*.md")
	if err != nil {
		return err
	}

	for _, file := range files {
		slug := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		// Generate as post/slug/index.html for GitHub Pages clean URLs
		postPath := filepath.Join("post", slug, "index.html")
		reqURL, _ := url.Parse("/post/" + slug)
		req := &http.Request{
			URL: reqURL,
		}
		if err := generatePage(outputDir, postPath, func(w http.ResponseWriter) error {
			PostHandler(w, req)
			return nil
		}); err != nil {
			return err
		}
	}

	// Generate post pages from thoughts directory
	thoughtsFiles, err := filepath.Glob("thoughts/*.md")
	if err != nil {
		return err
	}

	for _, file := range thoughtsFiles {
		slug := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		// Generate as post/slug/index.html for GitHub Pages clean URLs
		postPath := filepath.Join("post", slug, "index.html")
		reqURL, _ := url.Parse("/post/" + slug)
		req := &http.Request{
			URL: reqURL,
		}
		if err := generatePage(outputDir, postPath, func(w http.ResponseWriter) error {
			PostHandler(w, req)
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

// generatePage generates a single HTML page by executing a handler
func generatePage(outputDir, filename string, handler func(http.ResponseWriter) error) error {
	var buf bytes.Buffer
	w := &responseWriter{ResponseWriter: &mockResponseWriter{buf: &buf}}

	if err := handler(w); err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, filename)
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// mockResponseWriter is a minimal http.ResponseWriter implementation
type mockResponseWriter struct {
	buf    *bytes.Buffer
	header http.Header
	status int
}

func (m *mockResponseWriter) Header() http.Header {
	if m.header == nil {
		m.header = make(http.Header)
	}
	return m.header
}

func (m *mockResponseWriter) Write(b []byte) (int, error) {
	return m.buf.Write(b)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.status = statusCode
}

// responseWriter wraps a ResponseWriter to capture output
type responseWriter struct {
	http.ResponseWriter
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	slug := r.URL.Path[len("/post/"):]
	post, err := LoadPost(slug)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "post.gohtml")))
	if err := tmpl.Execute(w, post); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}

func LoadPost(slug string) (PostData, error) {
	// Try to find the post in posts directory first
	file := filepath.Join("posts", slug+".md")
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// If not found, try thoughts directory
		file = filepath.Join("thoughts", slug+".md")
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return PostData{}, err
		}
	}

	// Load post using the helper function that parses frontmatter
	return LoadPostFromFile(file)
}

// postsHandler is a generic handler for rendering pages with blog posts.
func postsHandler(w http.ResponseWriter, loadPosts func() ([]PostData, error), title, templateName string) {
	posts, err := loadPosts()
	if err != nil {
		http.Error(w, "Error loading posts", http.StatusInternalServerError)
		log.Printf("Error loading posts for %s: %v", title, err)
		return
	}

	data := TemplateData{
		Title: title,
		Posts: posts,
	}

	tmpl := template.Must(template.ParseFiles(
		filepath.Join("templates", "base.gohtml"),
		filepath.Join("templates", templateName),
	))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		log.Printf("Error executing template %s: %v", templateName, err)
		return
	}
}

// HomeHandler renders the home page with blog posts.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	postsHandler(w, LoadBlogPosts, "Home", "home.gohtml")
}

// ThoughtsHandler renders the thoughts blog posts page.
func ThoughtsHandler(w http.ResponseWriter, r *http.Request) {
	postsHandler(w, LoadThoughtsPosts, "Thoughts", "thoughts.gohtml")
}

// AboutHandler serves the About page.
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	content, err := RenderMarkdown("nav/about.md")
	if err != nil {
		http.Error(w, "Error loading about page", http.StatusInternalServerError)
		return
	}
	data := struct {
		Title   string
		Content template.HTML
	}{
		Title:   "About Me",
		Content: content,
	}
	tmpl := template.Must(template.ParseFiles(filepath.Join("templates", "base.gohtml"), filepath.Join("templates", "about.gohtml")))
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
		return
	}
}
