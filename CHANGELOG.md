# Changelog

## [0.7.2](https://github.com/janosmiko/vau/compare/v0.7.1...v0.7.2) (2026-09-24)


### Bug Fixes

* build copy-as-YAML output with yaml.v3 ([06c157a](https://github.com/janosmiko/vau/commit/06c157af3c01a0938af60949ba56425d4f16e7bc))
* cache 4xx kv probe errors and cancel the probe with ctx ([d3432cb](https://github.com/janosmiko/vau/commit/d3432cb560269fe1348db0cd8adcead2e6c48773))
* cancel list requests and retry failed kv version probes ([0de68bc](https://github.com/janosmiko/vau/commit/0de68bc240ad59bb71cca7c63ee27afa4b9e06e6))
* capture parent path before listParent goroutine reads it ([d436adf](https://github.com/janosmiko/vau/commit/d436adf211079bccbe568675ce3a961506a3c3fd))
* capture selection before renameEntry's async cmd runs ([b15e44e](https://github.com/janosmiko/vau/commit/b15e44e272a4f3d664d7983695ac02a625836e50))
* capture vault client before returning tea.Cmd closures ([b4768ac](https://github.com/janosmiko/vau/commit/b4768acd93ce04678bfb5982221cc9b893f16959))
* capture vault client in secret operation closures ([30aa38d](https://github.com/janosmiko/vau/commit/30aa38d0b856b791c2ab1b363fafd134a3239c65))
* clamp json preview capacity for tiny terminals ([2c66848](https://github.com/janosmiko/vau/commit/2c6684812b3b6f3df2f031463c54c5a795a64021))
* clear stale yanked data when a new single yank starts ([1075147](https://github.com/janosmiko/vau/commit/1075147a15e5636dde03191be50ae949547a0ac9))
* delete the selected bookmark by slot ([26aa44d](https://github.com/janosmiko/vau/commit/26aa44d7ebbda32a1012bb72572e3c5345781d17))
* delete the selected bookmark when slots are empty ([ee77743](https://github.com/janosmiko/vau/commit/ee777430ac2a9494018750fe3fe1315e79ad0b00))
* drop list, secret and version results from a previous mount ([9870599](https://github.com/janosmiko/vau/commit/98705993507ca685e290050b738a9f11dd25dce9))
* drop preview results that outrace the current selection ([f69ab29](https://github.com/janosmiko/vau/commit/f69ab2986ad4fe2dfbda3a7f812c1cfb3b1151b4))
* drop stale yank, history and new-secret results ([451f25c](https://github.com/janosmiko/vau/commit/451f25c084bab5132fdd4f5886ccf3e556b6e824))
* let mount-level mouse reach access categories ([0d211c9](https://github.com/janosmiko/vau/commit/0d211c9502ca4ebcf2059b32a6377138555693af))
* make vault client mount immutable per value ([84f2b2f](https://github.com/janosmiko/vau/commit/84f2b2f6a5e68d75e96464bae5288a372c3d8798))
* move CountRecursive off the Update goroutine ([111dc70](https://github.com/janosmiko/vau/commit/111dc700bdc7b32a6cae669c1ff9f8146c379f5b))
* move jump-path completion lookup off the Update goroutine ([02dcf32](https://github.com/janosmiko/vau/commit/02dcf32cc554eef9fa3d8a9888a6b093b8e3a046))
* persist bookmark changes before applying them ([59df5c6](https://github.com/janosmiko/vau/commit/59df5c6221a4ee5a5e33fbadba635b2c592e552f))
* redo single-secret cut-paste in the right direction ([70c0ab6](https://github.com/janosmiko/vau/commit/70c0ab6de490d0f86b7be491fbdb512eb22cb8b2))
* reject inline key rename that collides with existing key ([0566741](https://github.com/janosmiko/vau/commit/0566741efee8690050eeaef6ee02e4b687c3c330))
* reject mark slots unreachable from bookmark overlay ([be0fbda](https://github.com/janosmiko/vau/commit/be0fbdab57a91337fe754636c2408a5e432f4f43))
* reject single-secret paste across mounts ([5822c59](https://github.com/janosmiko/vau/commit/5822c59fd78d6b85a86e505ccf6ab4eed752bfcf))
* release progress context when an operation finishes ([c7eabe6](https://github.com/janosmiko/vau/commit/c7eabe6ccdf39fb5f07e8b8e5baedc722c21ba0b))
* restore explorer and role list key precedence ([306bbca](https://github.com/janosmiko/vau/commit/306bbcaea2f5eaf3f92210b05ff9e2524690f3c2))
* run the editor through sh so quoted values work ([502333a](https://github.com/janosmiko/vau/commit/502333a8ff26424c35b7d45d1a5a0e4aee90587c))
* run undo and redo on the mount the action came from ([45fb1c9](https://github.com/janosmiko/vau/commit/45fb1c902202ccde5b133dcbec8da0d2a74f5ff3))
* snapshot secret state before inline edit for undo/esc ([9f1b44e](https://github.com/janosmiko/vau/commit/9f1b44e2e4c996dbcc74cc95a6dd240d1e8f2ec3))
* stop recursive count when the operation is cancelled ([644e043](https://github.com/janosmiko/vau/commit/644e043cc12d49d1d25642b28f5c76c5f15f8ad1))
* support editor values that include arguments ([c933d8a](https://github.com/janosmiko/vau/commit/c933d8acd07a2442ebfda104e99a48e912fe0561))
* undo the paste destination the loop actually chose ([c63ccf3](https://github.com/janosmiko/vau/commit/c63ccf3d2928200d430f95576175fe618eaead07))
* write restored secret before deleting the cut copy ([e036e50](https://github.com/janosmiko/vau/commit/e036e501eee151332dc2e8aa901c0a6024a07e00))

## [0.7.1](https://github.com/janosmiko/vau/compare/v0.7.0...v0.7.1) (2026-09-24)


### Features

* add dockerconfigjson credential view ([d519014](https://github.com/janosmiko/vau/commit/d51901432e762370c0b0771f4bb0a812c9f42ba9))
* add dockerconfigjson credential view ([1dd20cb](https://github.com/janosmiko/vau/commit/1dd20cbd439ed7a2ea38f6c466ead5a9414e365b))


### Bug Fixes

* bump go and deps to clear govulncheck findings ([c4a083b](https://github.com/janosmiko/vau/commit/c4a083b27d1eaa7b79b06a7a95ed40c7184a0e6b))
* bump go and deps to clear govulncheck findings ([cc932e2](https://github.com/janosmiko/vau/commit/cc932e270176bf6e3e221f9629f80bac35648fe5))
* remove blank row below help bar ([72948e9](https://github.com/janosmiko/vau/commit/72948e926a77f47bf908d7fca0827909a04ed5c5))
* remove blank row below help bar ([2283532](https://github.com/janosmiko/vau/commit/22835320f3c846da693b57ba35ce5f6ad2f1988e))
