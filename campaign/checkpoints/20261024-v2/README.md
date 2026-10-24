# Remote recovery checkpoint, 20261024-v2

SOURCE ONLY. UNAPPLIED. UNACCEPTED. Coordinator review is required. These frozen alternatives are not a coalesced solution. Do not apply all patches together blindly. SAM v1 is WIP: DO NOT APPLY AS VERIFIED.

Public214 base: Git `6586fdea6a69384390021bc99f628d33082407e9`, 4,525 files, source-manifest SHA256 `ea70963a866cd310e10a60007234bd38ee26740167fcace5bdcb589077185adc`. Full base and candidate manifests retain their original logical paths and bytes. Every source file was recomputed without path filters. The base also matches every path, Git blob and mode in the exact Git commit. DELTA.json lists every added/modified/deleted path, pre/post SHA256, Git blob identity, modes, sizes and explicit logical-path to stored-postimage mapping.

Postimages live only in content-addressed `blobs/sha256/HASH.blob` files. This checkpoint has zero .go or .java files and adds zero Go packages. Never blindly copy a blob directory as source. Patches remain .patch data. Full source manifests retain original .go/.java logical paths. `restore_postimages.py` uses only the explicit mapping and checks the full base and candidate manifests. Each postimage is written to a unique temporary inode in the destination directory, chmodded there and installed with os.replace; it never writes into the existing destination inode. Its finally block cleans up only its own temporary path. This preserves hardlinked donor snapshots.

| Alternative | Full source-manifest SHA256 | Files | Handoff status |
| --- | --- | ---: | --- |
| union-metadata-bounds-kind | `c54e8a5cab4b65b90741419e1ada47f4630cc44ae559e276e875bbe6c590e444` | 4530 | UNACCEPTED_STRICT3_FAILED_300 |
| raf-file-overload-v10 | `b2d2e838ae21d617751f47acaef4f21aeb0c0fb77e1e9bdc7628d3ef9c938c43` | 4530 | UNACCEPTED_V10_STOP_NEW_FIXTURE_PREMISES |
| sam-anonymous-functional-v1 | `5acbad4c43c5e2c5bd56cf642caf6fe6ba6a005df7a5d0ea73e7065beb5ffbc9` | 4529 | WIP_HOLD_V1_SUPERSEDED_DO_NOT_APPLY_AS_VERIFIED |

Current coordinator status: union strict3 FAILED300 (actual c171a30e); RAF V10 STOP on new fixture premises, old22 pass (fc2d3b2e), with fixture-only V11 ready; SAM V1 HOLD is superseded by SAM candidate V2, which remains HOLD for a local named class inside anonymous context. Only the original three frozen V1-checkpoint source candidates are captured here. RAF V11 and SAM candidate V2 source are NOT included. Captured SAM V1 retains its known provenance and LocalNames registry defects. The packager did not interpret execution outputs. STATUS.json records owners, review IDs, failure stages and hashes of referenced local plans. All pending independent source/controller reviews, strict output decoding, fresh registry/JVM/native/whole-program/normal/race/target gates, lifecycle cleanup and Root acceptance remain required under the original plans. Plans and verbose artifacts are not repackaged; obtain and independently re-pin them before execution. No payload lease or Netty parity/completion, performance gain or all-Java claim is granted.

Map42 code and full-manifest payload are NOT INCLUDED. NOT-INCLUDED.json records its exact local source/manifest path and SHA, 4,647-file source, private 4,646-file base, HOLD reasons and 502-path public214 divergence. Its broad foreign-base source/CI/campaign replacements and removals must not be treated as a public214 regional patch. Map43 is not captured.

## Restore one alternative for review

Keep the bundle outside the fresh source checkout. Check MANIFEST.json SHA256 against the coordinator checkpoint record, then run `python3 /absolute/path/to/bundle/verify.py`. MANIFEST.json covers every other bundle file; its own digest is external to avoid self-reference.

Create a fresh clone or detached checkout of exact Git base `6586fdea6a69384390021bc99f628d33082407e9` in an empty directory. Do not reuse active source or copy user untracked files. Verify the complete checkout, including absence of unexpected files:

```sh
python3 /absolute/path/to/bundle/verify.py --source /absolute/path/to/fresh214
```

Choose ONE candidate ID from the table and inspect STATUS.json, DELTA.json and patches first. For review-only reconstruction (known-broken SAM only when intentionally inspecting WIP):

```sh
git -C /absolute/path/to/fresh214 apply --check /absolute/path/to/bundle/candidates/CHOSEN-ID/FORWARD.patch
git -C /absolute/path/to/fresh214 apply /absolute/path/to/bundle/candidates/CHOSEN-ID/FORWARD.patch
python3 /absolute/path/to/bundle/verify.py --source /absolute/path/to/fresh214 --candidate CHOSEN-ID
```

For source directories outside a Git worktree, run git apply with repository discovery disabled through an appropriate GIT_CEILING_DIRECTORIES. Running Git in a nested directory of another repository can silently skip paths. Complete manifest verification after application is mandatory. To return this disposable reconstruction to public214, apply its INVERSE.patch and rerun the base verifier. Never overwrite later work with inverse patches or postimages.

An alternative reconstruction applies the explicit blob mapping to another verified fresh public214 copy:

```sh
python3 /absolute/path/to/bundle/restore_postimages.py --candidate CHOSEN-ID --target /absolute/path/to/fresh214
```

This refuses non-base source and checks all restored bytes against the full candidate manifest. SAM reconstruction remains unverified WIP. Qualification requires the original repair/review/lease process.

Inherited PATCH-INTEGRITY-PROOF.json files record original checkpoint V1 byte-integrity and V1 utility executions; they are unchanged historical proofs. V2-HARDLINK-RESTORE-PROOF.json records actual execution of the current atomic utility on a disposable public214 target initially hardlinked to all 4,525 files of an independent disposable sourcebase. The target matched the complete union candidate while the complete donor base hashes and modes, and frozen public214 hashes, remained unchanged. All five modified destination inodes were replaced; no temporary files remained. All 37 original source manifests, patches and blobs remain byte-exact from V1. V1 and its manifest SHA256 `5ba11b30aa6986206bc2a0b3d2b307389f764d3869a9f33a5585a523071d2f64` are preserved.

The original PATCH-INTEGRITY-PROOF.json records actual forward application in a fresh full base copy, explicit inverse application, reverse-forward application in a separate fresh full candidate copy, and mapping restoration in fresh full base copies, including execution of the shipped restore/verify utilities. All complete manifests matched. Copies used distinct inodes, frozen inputs were rehashed, and scratch copies were removed. These are byte-recovery proofs only: no Go/JVM/transpiler execution, tests, new oracles, Git refs/commits/pushes, active source or user-file changes were made. Campaign docs/state and every other unchanged file were excluded from postimages only because their actual source bytes were unchanged.

## Complete source delta paths

### union-metadata-bounds-kind

- `transpiler/expression.go`
- `transpiler/generic_method_dispatch.go`
- `transpiler/source_generic_demand_kind211_test.go`
- `transpiler/source_generic_demand_syntax.go`
- `transpiler/source_generic_metadata209_test.go`
- `transpiler/source_generic_metadata209_transfer_test.go`
- `transpiler/source_generic_view_demand.go`
- `transpiler/source_identifier_bounds214_reference_test.go`
- `transpiler/source_identifier_bounds214_test.go`
- `transpiler/source_identifier_syntax_inventory.go`

### raf-file-overload-v10

- `stdjava/random_access_file_file_constructor209.go`
- `stdjava/random_access_file_file_constructor209_test.go`
- `stdjava/random_access_file_reference.go`
- `transpiler/intrinsics_random_access_file_reference.go`
- `transpiler/random_access_file_allocation_origin.go`
- `transpiler/random_access_file_file_constructor209.go`
- `transpiler/random_access_file_file_constructor209_probe_test.go`
- `transpiler/random_access_file_file_constructor209_test.go`
- `transpiler/random_access_file_reference_campaign_test.go`

### sam-anonymous-functional-v1

- `transpiler/anonymous_functional209_tdd_test.go`
- `transpiler/anonymous_functional213_input_tdd_test.go`
- `transpiler/anonymous_functional214_guard_test.go`
- `transpiler/anonymous_functional_source_contract.go`
- `transpiler/expression.go`
- `transpiler/functional_source_views.go`
- `transpiler/supplier_source_contract.go`
- `transpiler/tree_sitter.go`

