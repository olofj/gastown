#!/usr/bin/env bash
# blog/build.sh — Build the Gas Town blog from markdown sources.
#
# Renders markdown posts into static HTML using templates.
# Output goes to blog/public/ which nginx serves directly.
#
# Dependencies: bash, sed, awk (no external tools required)
#
# Usage: ./build.sh [--clean]

set -euo pipefail

BLOG_DIR="$(cd "$(dirname "$0")" && pwd)"
CONTENT_DIR="$BLOG_DIR/content/posts"
TEMPLATE_DIR="$BLOG_DIR/templates"
STATIC_DIR="$BLOG_DIR/static"
PUBLIC_DIR="$BLOG_DIR/public"

# --- Argument parsing ---------------------------------------------------------

if [[ "${1:-}" == "--clean" ]]; then
  rm -rf "$PUBLIC_DIR"
  echo "[blog] Cleaned public/"
fi

# --- Setup output directories ------------------------------------------------

mkdir -p "$PUBLIC_DIR/posts" "$PUBLIC_DIR/css"

# --- Copy static assets ------------------------------------------------------

cp -r "$STATIC_DIR/css/"* "$PUBLIC_DIR/css/"

# Copy 404 page if it exists
if [[ -f "$BLOG_DIR/content/404.html" ]]; then
  cp "$BLOG_DIR/content/404.html" "$PUBLIC_DIR/404.html"
fi

# --- Template substitution (safe with & in replacement) ----------------------

# Bash parameter expansion treats & in replacement as the matched string.
# Use awk for safe substitution.
template_sub() {
  local template="$1"
  local placeholder="$2"
  local replacement="$3"
  awk -v pat="$placeholder" -v rep="$replacement" '{
    idx = index($0, pat)
    while (idx > 0) {
      $0 = substr($0, 1, idx-1) rep substr($0, idx+length(pat))
      idx = index($0, pat)
    }
    print
  }' <<< "$template"
}

# --- Parse markdown frontmatter -----------------------------------------------

parse_frontmatter() {
  local file="$1"
  local key="$2"
  sed -n '/^---$/,/^---$/p' "$file" | grep "^${key}:" | sed "s/^${key}: *//"
}

# --- Render markdown to simple HTML -------------------------------------------
# Minimal markdown-to-HTML: headings, paragraphs, code blocks, inline code,
# bold, italic, links, and lists. No external dependencies.

render_markdown() {
  local file="$1"
  local in_body=false
  local past_frontmatter=0
  local in_code_block=false
  local in_list=false
  local result=""

  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip frontmatter
    if [[ "$line" == "---" ]]; then
      past_frontmatter=$((past_frontmatter + 1))
      continue
    fi
    if [[ $past_frontmatter -lt 2 ]]; then
      continue
    fi

    # Code blocks
    if [[ "$line" =~ ^\`\`\` ]]; then
      if $in_code_block; then
        result+="</code></pre>"$'\n'
        in_code_block=false
      else
        result+="<pre><code>"$'\n'
        in_code_block=true
      fi
      continue
    fi
    if $in_code_block; then
      # Escape HTML in code blocks
      line="${line//&/&amp;}"
      line="${line//</&lt;}"
      line="${line//>/&gt;}"
      result+="$line"$'\n'
      continue
    fi

    # Empty line: close list if open, skip
    if [[ -z "$line" ]]; then
      if $in_list; then
        result+="</ul>"$'\n'
        in_list=false
      fi
      continue
    fi

    # Headings
    if [[ "$line" =~ ^###\  ]]; then
      local heading="${line#\#\#\# }"
      result+="<h3>$heading</h3>"$'\n'
      continue
    fi
    if [[ "$line" =~ ^##\  ]]; then
      local heading="${line#\#\# }"
      result+="<h2>$heading</h2>"$'\n'
      continue
    fi
    if [[ "$line" =~ ^#\  ]]; then
      local heading="${line#\# }"
      result+="<h1>$heading</h1>"$'\n'
      continue
    fi

    # List items
    if [[ "$line" =~ ^-\  ]]; then
      if ! $in_list; then
        result+="<ul>"$'\n'
        in_list=true
      fi
      local item="${line#- }"
      item="$(inline_format "$item")"
      result+="<li>$item</li>"$'\n'
      continue
    fi

    # Close list if we hit a non-list line
    if $in_list; then
      result+="</ul>"$'\n'
      in_list=false
    fi

    # Paragraph
    line="$(inline_format "$line")"
    result+="<p>$line</p>"$'\n'
  done < "$file"

  # Close any open list
  if $in_list; then
    result+="</ul>"$'\n'
  fi

  echo "$result"
}

# --- Inline formatting --------------------------------------------------------

inline_format() {
  local text="$1"
  # Inline code
  text="$(echo "$text" | sed 's/`\([^`]*\)`/<code>\1<\/code>/g')"
  # Bold
  text="$(echo "$text" | sed 's/\*\*\([^*]*\)\*\*/<strong>\1<\/strong>/g')"
  # Italic
  text="$(echo "$text" | sed 's/\*\([^*]*\)\*/<em>\1<\/em>/g')"
  # Links
  text="$(echo "$text" | sed 's/\[\([^]]*\)\](\([^)]*\))/<a href="\2">\1<\/a>/g')"
  echo "$text"
}

# --- Format date --------------------------------------------------------------

format_date() {
  local date="$1"
  # Convert YYYY-MM-DD to human-readable
  if command -v date &>/dev/null; then
    date -d "$date" '+%B %d, %Y' 2>/dev/null || echo "$date"
  else
    echo "$date"
  fi
}

# --- Build posts and collect metadata -----------------------------------------

declare -a POST_ENTRIES=()

for post_file in "$CONTENT_DIR"/*.md; do
  [[ -f "$post_file" ]] || continue

  slug="$(basename "$post_file" .md)"
  title="$(parse_frontmatter "$post_file" "title")"
  date="$(parse_frontmatter "$post_file" "date")"
  summary="$(parse_frontmatter "$post_file" "summary")"
  date_formatted="$(format_date "$date")"

  # Render post body
  body="$(render_markdown "$post_file")"

  # Read post template and substitute
  post_template="$(cat "$TEMPLATE_DIR/post.html")"
  post_content="$(template_sub "$post_template" '{{.Title}}' "$title")"
  post_content="$(template_sub "$post_content" '{{.Date}}' "$date")"
  post_content="$(template_sub "$post_content" '{{.DateFormatted}}' "$date_formatted")"
  post_content="$(template_sub "$post_content" '{{.Body}}' "$body")"

  # Wrap in base template
  base_template="$(cat "$TEMPLATE_DIR/base.html")"
  page="$(template_sub "$base_template" '{{.Title}}' "$title")"
  page="$(template_sub "$page" '{{.Content}}' "$post_content")"

  echo "$page" > "$PUBLIC_DIR/posts/${slug}.html"

  # Collect for index
  POST_ENTRIES+=("${date}|${slug}|${title}|${date_formatted}|${summary}")
done

# --- Sort posts by date (newest first) and build index ------------------------

index_posts=""
while IFS= read -r entry; do
  [[ -z "$entry" ]] && continue
  IFS='|' read -r date slug title date_formatted summary <<< "$entry"

  index_posts+="<article class=\"post-summary\">"
  index_posts+="<h2><a href=\"/posts/${slug}.html\">${title}</a></h2>"
  index_posts+="<time datetime=\"${date}\">${date_formatted}</time>"
  if [[ -n "$summary" ]]; then
    index_posts+="<p>${summary}</p>"
  fi
  index_posts+="</article>"
done < <(printf '%s\n' "${POST_ENTRIES[@]}" | sort -r)

# Build index page
index_template="$(cat "$TEMPLATE_DIR/index.html")"
# Replace the range block with rendered posts
index_content="<h1>Gas Town Blog</h1>"
index_content+="<p class=\"subtitle\">Notes on building autonomous multi-agent systems</p>"
index_content+="<div class=\"posts\">${index_posts}</div>"

base_template="$(cat "$TEMPLATE_DIR/base.html")"
page="$(template_sub "$base_template" '{{.Title}}' "Home")"
page="$(template_sub "$page" '{{.Content}}' "$index_content")"
echo "$page" > "$PUBLIC_DIR/index.html"

# --- Summary ------------------------------------------------------------------

post_count=$(find "$PUBLIC_DIR/posts" -name "*.html" 2>/dev/null | wc -l)
echo "[blog] Built ${post_count} post(s) to public/"
