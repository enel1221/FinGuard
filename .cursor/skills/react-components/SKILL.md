---
name: react-components
description: React component patterns for FinGuard frontend. Covers file organization, functional component conventions, MUI styling, state management, hooks, and loading/error/empty state handling. Use when creating React components, modifying frontend UI, adding pages, or working in web/frontend/src/.
---

# React Component Patterns

## File Organization

```
web/frontend/src/
  components/
    {FeatureName}/
      {ComponentName}.tsx       -- one component per file
      {SubComponent}.tsx        -- related sub-components in same folder
  lib/
    api.ts                      -- API client and shared types
    AppTheme.ts                 -- theme type definitions
    themes.ts                   -- MUI theme creation
  redux/
    store.ts                    -- Redux store (theme + sidebar only)
    {slice}Slice.ts             -- Redux slices
  plugin/
    index.ts                    -- plugin loader
```

Rules:
- One component per file. File name matches component name, PascalCase.
- Group related components in a feature folder: `components/Project/ProjectDetail.tsx`, `components/Project/ProjectMembers.tsx`.
- Path alias `@/` maps to `src/`. Use it for imports: `import { api } from '@/lib/api'`.

## Component Structure

Every component follows this template:

```tsx
import { useState, useEffect } from 'react';
import { Box, Typography, CircularProgress } from '@mui/material';
import { api, type SomeType } from '@/lib/api';

interface MyComponentProps {
  projectId: string;
}

export default function MyComponent({ projectId }: MyComponentProps) {
  const [data, setData] = useState<SomeType | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<SomeType>(`/projects/${projectId}`)
      .then(setData)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [projectId]);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" mt={8}>
        <CircularProgress color="primary" />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Title</Typography>
      {/* content */}
    </Box>
  );
}
```

Rules:
- `export default function ComponentName()` -- functional, default export, PascalCase.
- Define `interface ComponentNameProps` above the component. Never use `any`.
- No class components. Hooks only.

## Styling

MUI `sx` prop is the **only** styling approach. No CSS files, no CSS modules, no Tailwind, no inline style objects.

```tsx
// Correct
<Box sx={{ display: 'flex', p: 3, bgcolor: 'background.paper' }}>

// Wrong -- no raw HTML with inline styles
<div style={{ display: 'flex', padding: 24 }}>
```

Rules:
- Use MUI theme tokens: `primary.main`, `text.secondary`, `background.paper`, `divider`.
- Use MUI spacing shorthand: `p`, `m`, `mt`, `mb`, `px`, etc. (unit = 8px).
- Never hardcode colors. Always reference theme palette.
- For responsive layouts, use MUI `Grid2` with size breakpoints:

```tsx
<Grid container spacing={2}>
  <Grid size={{ xs: 12, sm: 6, md: 3 }}>
    <Card>...</Card>
  </Grid>
</Grid>
```

## MUI Component Preferences

Use MUI components over raw HTML:

| Instead of | Use |
|-----------|-----|
| `<div>` | `<Box>` |
| `<span>` / `<p>` / `<h1>` | `<Typography variant="...">` |
| `<button>` | `<Button>` |
| `<input>` | `<TextField>` |
| `<select>` | `<Select>` with `<MenuItem>` |
| `<table>` | `<Table>`, `<TableHead>`, `<TableBody>`, `<TableRow>`, `<TableCell>` |

Typography variants used in the project:
- `h5` with `fontWeight={600}` for page titles
- `h6` for section headings
- `caption` with `color="text.secondary"` for labels
- `body2` with `color="text.secondary"` for descriptions
- `h4` with `fontWeight={700}` for large stat numbers

## State Management

| Kind of state | Where |
|--------------|-------|
| Local UI state (form values, open/close) | `useState` |
| Server data (projects, costs, health) | `useState` + `useEffect` + `api` calls |
| Global UI state (theme, sidebar entries) | Redux (`@reduxjs/toolkit`) |

**Never put server data in Redux.** Redux is only for client-side UI state (theme preference, sidebar navigation entries).

Access Redux state:
```tsx
const currentTheme = useSelector((state: RootState) => state.theme.current);
const dispatch = useDispatch();
dispatch(setTheme('FinGuard Dark'));
```

## Loading / Error / Empty States

Every data-driven component must handle all three states:

```tsx
// Loading
if (loading) {
  return (
    <Box display="flex" justifyContent="center" mt={8}>
      <CircularProgress color="primary" />
    </Box>
  );
}

// Error (inline or via state)
// Catch API errors and show a message

// Empty
if (!items || items.length === 0) {
  return (
    <Card>
      <CardContent>
        <Typography color="text.secondary">
          No items found. Create one to get started.
        </Typography>
      </CardContent>
    </Card>
  );
}
```

## Hooks

- `useState` -- local state
- `useEffect` -- data fetching, side effects
- `useMemo` -- expensive computed values (e.g., theme creation)
- `useSelector` / `useDispatch` -- Redux
- `useNavigate`, `useParams`, `useLocation` -- React Router

Extract reusable logic into custom hooks in `src/hooks/` when the same pattern appears in 2+ components.

## Routing

React Router v6. Routes defined in `App.tsx`:

```tsx
<Route path="/things" element={<ThingList />} />
<Route path="/things/:thingId" element={<ThingDetail />} />
```

Use `useNavigate()` for programmatic navigation, `useParams()` for path params.

## Dialogs

For create/edit forms, use MUI `Dialog`:

```tsx
const [dialogOpen, setDialogOpen] = useState(false);

<Dialog open={dialogOpen} onClose={() => setDialogOpen(false)}>
  <DialogTitle>Create Thing</DialogTitle>
  <DialogContent>
    <TextField ... />
  </DialogContent>
  <DialogActions>
    <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
    <Button variant="contained" onClick={handleSave}>Save</Button>
  </DialogActions>
</Dialog>
```

## Imports

Order:
1. React (`react`, `react-dom`)
2. Third-party libraries (`@mui/*`, `react-router-dom`, `recharts`)
3. Internal modules (`@/lib/*`, `@/redux/*`)
4. Relative imports (sibling components)

## Testing Requirements

New components are not complete until they have passing tests. Follow the **testing** skill for patterns (Vitest, React Testing Library, `renderWithProviders`).

When adding or modifying a component:

1. Write a `ComponentName.test.tsx` file alongside the component.
2. Test user-visible behavior: loading states, rendered content, error states, and empty states.
3. Run `make test` and confirm all tests pass (backend + frontend).
4. Do not consider the feature implemented until `make test` exits cleanly.
