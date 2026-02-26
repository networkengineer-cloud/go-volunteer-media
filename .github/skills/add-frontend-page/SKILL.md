---
name: add-frontend-page
description: Add a new page or major feature to the React frontend of go-volunteer-media. Covers creating the page component and CSS, adding the route in App.tsx, adding navigation links, and writing a Playwright E2E test. Use when building new UI screens or significant user-facing workflows.
argument-hint: [page name or feature description]
---

# Add a New Frontend Page

Follow these steps in order.

## Step 1 — Create the page component (`frontend/src/pages/`)

Create `frontend/src/pages/FooPage.tsx` and `frontend/src/pages/FooPage.css`.

Component conventions:
- Functional component with TypeScript: `const FooPage: React.FC = () => { ... }`
- Import the co-located CSS: `import './FooPage.css'`
- Fetch data via API methods from `frontend/src/api/client.ts` — never call axios directly in the component
- Use custom hooks for non-trivial data-fetching logic; keep the component focused on rendering
- Handle loading, error, and empty states explicitly
- All interactive elements must have accessible labels (`aria-label`, `<label>`, etc.)
- Keyboard navigation must work without a mouse

Minimal scaffold:

```tsx
import React, { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { fooApi, Foo } from '../api/client';
import './FooPage.css';

const FooPage: React.FC = () => {
  const { groupId } = useParams<{ groupId: string }>();
  const parsedGroupId = parseInt(groupId ?? '', 10);
  if (isNaN(parsedGroupId)) return null; // route guard — should not happen with correct route config
  const [items, setItems] = useState<Foo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fooApi.getAll(parsedGroupId)
      .then(res => setItems(res.data))
      .catch(() => setError('Failed to load items'))
      .finally(() => setLoading(false));
  }, [parsedGroupId]);

  if (loading) return <div className="loading">Loading...</div>;
  if (error) return <div className="error" role="alert">{error}</div>;

  return (
    <main className="foo-page">
      <h1>Foo</h1>
      {/* ... */}
    </main>
  );
};

export default FooPage;
```

## Step 2 — Add the route (`frontend/src/App.tsx`)

Import the new page and add a `<Route>` inside the existing router. Routes are React Router v6:

```tsx
import FooPage from './pages/FooPage';

// Inside <Routes>:
<Route path="/groups/:groupId/foos" element={<FooPage />} />
```

Follow the existing path conventions — group-scoped pages use `/groups/:groupId/<feature>`.

## Step 3 — Add a navigation link (`frontend/src/components/Navigation.tsx`)

If the page should appear in the sidebar/nav, add a `<NavLink>`:

```tsx
<NavLink to={`/groups/${groupId}/foos`} className={({ isActive }) => isActive ? 'active' : ''}>
  Foo
</NavLink>
```

Group-admin-only links should be conditionally rendered: wrap in `{isGroupAdmin && <NavLink ...>}`.

## Step 4 — Write a Playwright E2E test

Create `frontend/tests/foo-page.spec.ts`. See the `playwright-e2e-test` skill for the full authoring guide.

At minimum, the spec should cover:
- Page renders with expected heading/content
- Core user action works (create, update, or delete)
- Error state is displayed when the API fails
- Accessibility: tab order, ARIA roles, visible focus

## Checklist

- [ ] `frontend/src/pages/FooPage.tsx` created
- [ ] `frontend/src/pages/FooPage.css` created
- [ ] Route added in `frontend/src/App.tsx`
- [ ] Navigation link added in `frontend/src/components/Navigation.tsx` (if applicable)
- [ ] Loading, error, and empty states handled
- [ ] Accessibility requirements met (labels, keyboard nav, focus indicators)
- [ ] Playwright E2E spec added in `frontend/tests/`
