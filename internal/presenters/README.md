# Presenters

The `presenters` package provides utilities for shaping and preparing data for presentation within the application.

These functions are generally **pure**. They do not perform network calls or side effects. Instead, they return data structures that are ready to be passed to other layers, such as for network responses.

The primary consumer of this package is the handler layer, which uses presenters to bridge the gap between core business logic and external interfaces like APIs.
