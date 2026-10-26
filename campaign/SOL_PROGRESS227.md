# Sol checkpoint 227: generic ABI regression guards

Two existing tests still asserted earlier generated Go APIs. The dependent-generic test now checks the source-owned physical widen call, its first-bound slots, identity witnesses and nominal Base consumer. The raw-generic test executes the unchanged Java fixture's generated Run method before checking erased fields, checked casts and execution-aware concatenation; its observer compares canonical JavaString values. The expected result remains `value=7`.

Both tests and the dependent-binder shadowing guard pass in fresh normal and race runs, with no failures or skips. Independent actual review: `b10b65320671cc9a5285bced3197058eccf3309598157ac12c54282aebbb16e8`. The exact raw Java fixture also compiled and ran under JDK21, printing `value=7`. Production code and application fixtures are unchanged. The old layout assertion failure and interim observer compile failure remain recorded.

Hosted CI for the new commit remains pending. This checkpoint does not establish full Gson or Netty compatibility; their application gates and the remaining compiler defects are active.
