# Travel Agent Skill

## Purpose

This file is the private reusable rulebook for the `travel` travel agent.

## Stable travel contract

- One city page represents one city-focused travel space.
- The Workspace detail view and standalone HTML city page must stay aligned to the same guide content.
- Page content should remain city-specific and must not mix multiple target cities into one page.
- Preserve structured guide fields so later revisions can update city pages without rebuilding from scratch.
- Every newly generated HTML travel page must remain usable in both desktop and mobile browsers; neither layout is optional.
- Before drafting the itinerary, first enumerate the researched recommendation pool for each active category instead of jumping straight to the final route plan.
- Treat `guide_html_url` as the canonical public deliverable for the HTML guide, but only surface it after the travel service has been successfully published.
- Generate or overwrite the current request's `index.html` in the session workspace root before publishing, and never publish a stale or unrelated page from another request or directory.
- If `index.html` is missing or publish fails, treat the HTML guide as blocked rather than claiming delivery.
- The public guide host must use `https://travel-<session_short_hash>.alter0.cn`; do not fall back to nested hosts such as `<session_short_hash>.travel.alter0.cn`.
- Travel guide publishing must stay certificate-safe by using a single-label subdomain under `*.alter0.cn`.

## Default content expectations

- Provide a clear city title and short summary.
- Recommendation sections should list the category pool before the day-by-day plan, so readers can review the full shortlist rather than only the places that made the final itinerary.
- Food and drink recommendations should separate snacks, breakfast, signature dishes, and signature drinks, and should mix time-honored shops with high-scoring Dianping picks when available.
- Sightseeing recommendations should be grouped by type such as parks, museums, performances, and other relevant attraction formats for the city.
- Accommodation recommendations should list popular hotels by budget range or stay tier, with useful area or district guidance when available.
- Treat those food, sightseeing, and accommodation groups as the default core structure, then flexibly add city-specific categories when the destination clearly calls for them, such as markets, river cruises, night views, hot springs, snow fields, or temple fairs.
- Each recommendation group should name its data source, such as Dianping high-score lists, official scenic-spot references, hotel booking platforms, or other explicit source labels used in the page.
- Keep visible sections for highlights, day-by-day route planning, metro or transit guidance, food recommendations, practical notes, and map-oriented hints when available.
- Prefer concise, scan-friendly sections that can be extended without breaking the page layout.
- Treat desktop and mobile as a shared acceptance bar: desktop should preserve hierarchy and reading rhythm, while mobile should keep single-column readability, touch-safe actions, and stable spacing without horizontal overflow.
- When the session workspace root already contains `index.html`, reuse that root as the travel guide artifact source instead of requiring a separate web build output.

## Store Here

- Durable travel-page structure, section ordering, tone, naming conventions, and stable rendering preferences.
- Reusable itinerary composition heuristics, transit defaults, food recommendation framing, and map-output conventions requested by the user.
- Stable travel-agent defaults that should apply across future city pages handled by this agent.

## Keep Out

- Repository or workspace operating rules that belong in `.alter0/agents/travel/AGENTS.md`.
- One-off trip constraints, temporary dates, current-session notes, or single-city exceptions that should stay in the target guide data.
- Shared repository policy or non-travel reusable behavior that should live outside this travel-agent skill.

## Editing Rules

- Apply focused updates instead of replacing the whole file.
- Preserve stable rules unless the user clearly changes a durable preference.
- Promote only reusable travel guidance into this file.
