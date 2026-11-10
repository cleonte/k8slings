# gatewayapi3: two conditions, two matches

In a route rule, `matches` is a list, and a request is sent to the rule's backends if it satisfies **any** of the matches (**or**). Inside one match, the conditions (`path`, `headers`, `method`, `queryParams`) must **all** hold (**and**). Listing a path and a header as two matches therefore means "this path, or this header", which sends far more traffic than meant.

This rule should send to the beta Service only the requests for `/api` that **also** carry the header `X-Team: blue`.
