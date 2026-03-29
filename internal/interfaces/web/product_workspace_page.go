package web

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	productdomain "alter0/internal/product/domain"
)

func (s *Server) productPublicPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	parts, ok := productPublicPageParts(r.URL.Path)
	if !ok || len(parts) != 3 {
		http.NotFound(w, r)
		return
	}
	if parts[1] != "spaces" {
		http.NotFound(w, r)
		return
	}
	productID := strings.TrimSpace(parts[0])
	spaceID := normalizeWorkspacePageSpaceID(parts[2])
	if spaceID == "" {
		http.NotFound(w, r)
		return
	}

	product, statusCode, err := s.resolvePublicProduct(productID)
	if err != nil {
		if statusCode == http.StatusNotFound {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, statusCode, map[string]string{"error": err.Error()})
		return
	}
	if !strings.EqualFold(product.ID, productdomain.TravelProductID) {
		http.NotFound(w, r)
		return
	}
	if s.travelGuides == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "travel guide service unavailable"})
		return
	}
	guide, found := s.travelGuides.GetGuide(spaceID)
	if !found {
		http.NotFound(w, r)
		return
	}

	page := renderTravelGuideHTMLPage(product, guide)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(page))
}

func productPublicPageParts(path string) ([]string, bool) {
	const prefix = "/products/"
	if !strings.HasPrefix(path, prefix) {
		return nil, false
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return nil, false
	}
	parts := strings.Split(trimmed, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			return nil, false
		}
		cleaned = append(cleaned, trimmedPart)
	}
	return cleaned, true
}

func normalizeWorkspacePageSpaceID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimSuffix(trimmed, ".html")
	return strings.TrimSpace(trimmed)
}

func renderTravelGuideHTMLPage(product productdomain.Product, guide productdomain.TravelGuide) string {
	title := strings.TrimSpace(guide.City)
	if title == "" {
		title = strings.TrimSpace(guide.ID)
	}
	if title == "" {
		title = "Travel Guide"
	}
	pageTitle := title + " | " + strings.TrimSpace(product.Name)
	metaDescription := summaryText(guide.Content, 160)
	if metaDescription == "" {
		metaDescription = title + " travel guide"
	}
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>` + html.EscapeString(pageTitle) + `</title>
  <meta name="description" content="` + html.EscapeString(metaDescription) + `">
  <style>
    :root{
      --bg:#f4efe6;
      --paper:#fffdf8;
      --ink:#1f2937;
      --muted:#6b7280;
      --line:#ded6c8;
      --accent:#0f766e;
      --accent-soft:#d8f0ec;
      --chip:#eef2ff;
      --shadow:0 20px 45px -30px rgba(31,41,55,.45);
    }
    *{box-sizing:border-box}
    body{
      margin:0;
      background:
        radial-gradient(circle at top left, rgba(15,118,110,.12), transparent 32%),
        linear-gradient(180deg, #f8f4ec 0%, var(--bg) 100%);
      color:var(--ink);
      font:16px/1.6 "Segoe UI","PingFang SC","Hiragino Sans GB","Microsoft YaHei",sans-serif;
    }
    .page{
      max-width:1120px;
      margin:0 auto;
      padding:32px 20px 80px;
    }
    .hero{
      display:grid;
      gap:18px;
      padding:28px;
      border:1px solid rgba(222,214,200,.85);
      border-radius:28px;
      background:linear-gradient(140deg, rgba(255,253,248,.96), rgba(247,242,232,.92));
      box-shadow:var(--shadow);
    }
    .eyebrow{
      display:flex;
      flex-wrap:wrap;
      gap:10px;
      color:var(--muted);
      font-size:12px;
      letter-spacing:.12em;
      text-transform:uppercase;
    }
    h1{
      margin:0;
      font-size:clamp(32px,5vw,52px);
      line-height:1.05;
    }
    .summary{
      margin:0;
      max-width:860px;
      color:#334155;
      font-size:18px;
    }
    .meta-grid{
      display:grid;
      grid-template-columns:repeat(auto-fit,minmax(150px,1fr));
      gap:12px;
    }
    .meta-card{
      padding:14px 16px;
      border:1px solid var(--line);
      border-radius:18px;
      background:rgba(255,255,255,.72);
    }
    .meta-card span{
      display:block;
      color:var(--muted);
      font-size:12px;
      text-transform:uppercase;
      letter-spacing:.08em;
    }
    .meta-card strong{
      display:block;
      margin-top:6px;
      font-size:18px;
    }
    .layout{
      display:grid;
      grid-template-columns:minmax(0,2fr) minmax(280px,1fr);
      gap:20px;
      margin-top:22px;
    }
    .section{
      padding:22px;
      border:1px solid rgba(222,214,200,.9);
      border-radius:24px;
      background:var(--paper);
      box-shadow:var(--shadow);
    }
    .section h2{
      margin:0 0 14px;
      font-size:18px;
      letter-spacing:.03em;
    }
    .section pre{
      margin:0;
      white-space:pre-wrap;
      word-break:break-word;
      font:14px/1.65 "Consolas","SFMono-Regular","Liberation Mono",monospace;
      color:#223046;
      background:#fbfaf7;
      border:1px solid #ebe3d6;
      border-radius:18px;
      padding:18px;
    }
    .stack{
      display:grid;
      gap:14px;
    }
    .route-card,.layer-card{
      padding:16px 18px;
      border-radius:18px;
      background:#fbfaf7;
      border:1px solid #ebe3d6;
    }
    .route-card strong,.layer-card strong{
      display:block;
      margin-bottom:6px;
      font-size:16px;
    }
    .chip-list{
      display:flex;
      flex-wrap:wrap;
      gap:8px;
    }
    .chip{
      padding:7px 12px;
      border-radius:999px;
      background:var(--chip);
      color:#334155;
      font-size:13px;
    }
    .aside{
      display:grid;
      gap:20px;
      align-content:start;
    }
    .accent{
      background:linear-gradient(180deg, #f3fbfa 0%, #ecf7f5 100%);
      border-color:#c8e5df;
    }
    .list{
      margin:0;
      padding-left:18px;
    }
    .list li + li{
      margin-top:8px;
    }
    .empty{
      margin:0;
      color:var(--muted);
    }
    .footer{
      margin-top:22px;
      color:var(--muted);
      font-size:13px;
      text-align:center;
    }
    @media (max-width: 900px){
      .layout{
        grid-template-columns:1fr;
      }
      .page{
        padding:18px 14px 56px;
      }
      .hero,.section{
        border-radius:20px;
        padding:18px;
      }
    }
  </style>
</head>
<body>
  <main class="page">
    <section class="hero">
      <div class="eyebrow">
        <span>` + html.EscapeString(strings.TrimSpace(product.Name)) + `</span>
        <span>` + html.EscapeString(strings.TrimSpace(product.ID)) + `</span>
        <span>HTML Page</span>
      </div>
      <div>
        <h1>` + html.EscapeString(title) + `</h1>
        <p class="summary">` + html.EscapeString(summaryText(guide.Content, 220)) + `</p>
      </div>
      <div class="meta-grid">
        <article class="meta-card"><span>Days</span><strong>` + html.EscapeString(fmt.Sprintf("%d", guide.Days)) + `</strong></article>
        <article class="meta-card"><span>Revision</span><strong>` + html.EscapeString(fmt.Sprintf("%d", guide.Revision)) + `</strong></article>
        <article class="meta-card"><span>Travel Style</span><strong>` + html.EscapeString(defaultHTMLValue(guide.TravelStyle)) + `</strong></article>
        <article class="meta-card"><span>Budget</span><strong>` + html.EscapeString(defaultHTMLValue(guide.Budget)) + `</strong></article>
        <article class="meta-card"><span>Updated</span><strong>` + html.EscapeString(formatHTMLTime(guide.UpdatedAt)) + `</strong></article>
      </div>
    </section>
    <section class="layout">
      <div class="stack">
        <section class="section">
          <h2>Page Content</h2>
          <pre>` + html.EscapeString(strings.TrimSpace(guide.Content)) + `</pre>
        </section>
        <section class="section">
          <h2>Daily Routes</h2>
          ` + renderTravelGuideRouteCards(guide.DailyRoutes) + `
        </section>
        <section class="section">
          <h2>Map Layers</h2>
          ` + renderTravelGuideLayerCards(guide.MapLayers) + `
        </section>
      </div>
      <aside class="aside">
        <section class="section accent">
          <h2>Highlights</h2>
          ` + renderTravelGuideChipList([]string{guide.City, guide.TravelStyle, guide.Budget}) + `
        </section>
        <section class="section">
          <h2>Must Visit</h2>
          ` + renderTravelGuideList(guide.MustVisit) + `
        </section>
        <section class="section">
          <h2>Additional Requirements</h2>
          ` + renderTravelGuideChipList(append(append([]string{}, guide.AdditionalRequirements...), guide.KeepConditions...)) + `
        </section>
        <section class="section">
          <h2>Notes</h2>
          ` + renderTravelGuideList(guide.Notes) + `
        </section>
      </aside>
    </section>
    <p class="footer">Generated from the ` + html.EscapeString(strings.TrimSpace(product.Name)) + ` workspace page.</p>
  </main>
</body>
</html>`
}

func renderTravelGuideRouteCards(items []productdomain.TravelDailyRoute) string {
	if len(items) == 0 {
		return `<p class="empty">No route details yet.</p>`
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, `<article class="route-card">
  <strong>`+html.EscapeString(fmt.Sprintf("Day %d", item.Day))+`</strong>
  <p>`+html.EscapeString(defaultHTMLValue(item.Theme))+`</p>
  <p>`+html.EscapeString(strings.Join(item.Stops, " -> "))+`</p>
  <p>`+html.EscapeString(strings.Join(item.Transit, " / "))+`</p>
</article>`)
	}
	return `<div class="stack">` + strings.Join(rows, "") + `</div>`
}

func renderTravelGuideLayerCards(items []productdomain.TravelMapLayer) string {
	if len(items) == 0 {
		return `<p class="empty">No map layers yet.</p>`
	}
	rows := make([]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, `<article class="layer-card">
  <strong>`+html.EscapeString(defaultHTMLValue(item.Label))+`</strong>
  <p>`+html.EscapeString(defaultHTMLValue(item.Description))+`</p>
</article>`)
	}
	return `<div class="stack">` + strings.Join(rows, "") + `</div>`
}

func renderTravelGuideChipList(items []string) string {
	filtered := filterHTMLValues(items)
	if len(filtered) == 0 {
		return `<p class="empty">-</p>`
	}
	chips := make([]string, 0, len(filtered))
	for _, item := range filtered {
		chips = append(chips, `<span class="chip">`+html.EscapeString(item)+`</span>`)
	}
	return `<div class="chip-list">` + strings.Join(chips, "") + `</div>`
}

func renderTravelGuideList(items []string) string {
	filtered := filterHTMLValues(items)
	if len(filtered) == 0 {
		return `<p class="empty">-</p>`
	}
	rows := make([]string, 0, len(filtered))
	for _, item := range filtered {
		rows = append(rows, `<li>`+html.EscapeString(item)+`</li>`)
	}
	return `<ul class="list">` + strings.Join(rows, "") + `</ul>`
}

func filterHTMLValues(items []string) []string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	return values
}

func defaultHTMLValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}

func formatHTMLTime(value any) string {
	switch item := value.(type) {
	case interface {
		IsZero() bool
		Format(string) string
	}:
		if item.IsZero() {
			return "-"
		}
		return item.Format("2006-01-02 15:04:05 MST")
	default:
		return "-"
	}
}
