# ingress1: a path that matches only itself

Every path rule of an Ingress has a `pathType`, and it decides what the path matches. **`Exact`** matches that one path and nothing else: `/shop` is served, `/shop/` and `/shop/cart` get a 404. **`Prefix`** matches the path and everything below it, element by element: `/shop` serves `/shop`, `/shop/` and `/shop/cart`, but not `/shopping`.

This Ingress is meant to serve the whole shop application: its pages are all below `/shop`.
