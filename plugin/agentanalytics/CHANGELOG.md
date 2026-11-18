# Changelog

## [0.1.1](https://github.com/google/adk-go/compare/plugin/agentanalytics/v0.1.0...plugin/agentanalytics/v0.1.1) (2026-11-18)


### Bug Fixes

* **agentanalytics:** align checkpoint gating, MCP isError tool results, and empty error messages ([#2047](https://github.com/google/adk-go/issues/2047)) ([#2178](https://github.com/google/adk-go/issues/2178)) ([f423f02](https://github.com/google/adk-go/commit/f423f0299b5d64be0b3ff171c245cb2a1180839a))
* **agentanalytics:** cap response summary and function name summary length ([#2246](https://github.com/google/adk-go/issues/2246)) ([f55660a](https://github.com/google/adk-go/commit/f55660a66749dd7ffc80a25f4ac6c64bdb890817))
* **agentanalytics:** isolate BigQuery RowErrors and track RowErrorDrops ([#2376](https://github.com/google/adk-go/issues/2376)) ([b019ecc](https://github.com/google/adk-go/commit/b019ecc3f628a97c5b64f930a0c5a43f6182a8ce))
* **agentanalytics:** match an error-ended tool call to its own attempt ([#2377](https://github.com/google/adk-go/issues/2377)) ([2924008](https://github.com/google/adk-go/commit/2924008fb6269ec37419ce31a182f16e8c043b90))
* **agentanalytics:** preserve retired late parent cancel cause and custom metadata truncation flag ([#2020](https://github.com/google/adk-go/issues/2020)) ([ca197aa](https://github.com/google/adk-go/commit/ca197aaaa582e9c6d85da4559729a099a7c98907))
* **agentanalytics:** redact and bound code part labels in summaries ([#2332](https://github.com/google/adk-go/issues/2332)) ([cf55159](https://github.com/google/adk-go/commit/cf55159d2d7894e682fb006267910a8ded36ccb6)), closes [#2259](https://github.com/google/adk-go/issues/2259)
* **agentanalytics:** redact compound and state-prefixed sensitive keys ([#2250](https://github.com/google/adk-go/issues/2250)) ([6e73acc](https://github.com/google/adk-go/commit/6e73accaf9b3b333d050b7a6889a3c7a2bc8431f))
* **agentanalytics:** redact credential-named types and guard omitzero in JSONFormatter ([#2249](https://github.com/google/adk-go/issues/2249)) ([5b3d897](https://github.com/google/adk-go/commit/5b3d897b19898a31df19ab786ee85670e69e9a61))
* **agentanalytics:** sanitize unsafe GCSPathPrefix values and cap GCS object names at 1,024 bytes ([#2018](https://github.com/google/adk-go/issues/2018)) ([11f4e29](https://github.com/google/adk-go/commit/11f4e29f588ee0f08d2e372f76737412075460bf))
* **agentanalytics:** scope lookalike parent resolution to active goroutine ([#2379](https://github.com/google/adk-go/issues/2379)) ([6730502](https://github.com/google/adk-go/commit/673050266d9bf329f8cc46a93720ab27ba14ca46))


### Documentation

* **agentanalytics:** add package godoc documenting Go design and parity behavior ([#2176](https://github.com/google/adk-go/issues/2176)) ([1edb057](https://github.com/google/adk-go/commit/1edb0572264d71b3ba3ab4e9bd24a979e267a96e))
