export const REACT_MANAGED_CONTROL_ROUTES = [
  "skills",
  "cron-jobs",
] as const;

export type ReactManagedControlRoute = (typeof REACT_MANAGED_CONTROL_ROUTES)[number];

export const REACT_MANAGED_ROUTE_BODIES = [
  "settings",
] as const;

export type ReactManagedRouteBodyRoute = (typeof REACT_MANAGED_ROUTE_BODIES)[number];

export function getReactManagedControlRoutes(): ReactManagedControlRoute[] {
  return [...REACT_MANAGED_CONTROL_ROUTES];
}

export function getReactManagedRouteBodyRoutes(): ReactManagedRouteBodyRoute[] {
  return [...REACT_MANAGED_ROUTE_BODIES];
}

export function isReactManagedControlRoute(route: string): route is ReactManagedControlRoute {
  return REACT_MANAGED_CONTROL_ROUTES.includes(route as ReactManagedControlRoute);
}

export function isReactManagedRouteBody(route: string): route is ReactManagedRouteBodyRoute {
  return REACT_MANAGED_ROUTE_BODIES.includes(route as ReactManagedRouteBodyRoute);
}
