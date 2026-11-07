# lifecycle4: init containers run in the order they are written

Init containers run **one at a time, in the order of the list**, and each must finish before the next starts. When the second one needs what the first leaves behind, the order of the list is the only thing that makes it work. A list in the wrong order is not rejected: the first container to run simply finds nothing, and fails.

Here `fetch` downloads the template into a shared volume, and `render` fills it in. `render` is listed first.
