# Changelog

## [0.0.4](https://github.com/jycamier/retrotro/compare/retrotro-backend-v0.0.3...retrotro-backend-v0.0.4) (2026-02-25)


### Features

* add detailed logging for item creation and WebSocket broadcasting ([41d5d45](https://github.com/jycamier/retrotro/commit/41d5d45523d57f019a260676f0f6c32996bb870a))
* add drag & drop and assignee management to team actions Kanban ([0140af3](https://github.com/jycamier/retrotro/commit/0140af305f3beb703ef0177fe08faadfc0ac9baf))
* add Lean Coffee session ([#14](https://github.com/jycamier/retrotro/issues/14)) ([6930f31](https://github.com/jycamier/retrotro/commit/6930f3133c53e77188f83d970d0ddd545925dbe4))
* add NATS credentials file authentication support ([6d94855](https://github.com/jycamier/retrotro/commit/6d948555f77275c5a550d818091c40ae1e537cde))
* add nats secrets management ([0afcce3](https://github.com/jycamier/retrotro/commit/0afcce3a578c6d4535d9effe32ead93ddd8a4428))
* add pgbridge for events ([b90eebf](https://github.com/jycamier/retrotro/commit/b90eebfc0c4a1828c3b7f5487984f08c0af2672c))
* add team actions Kanban view ([2c3d3c2](https://github.com/jycamier/retrotro/commit/2c3d3c221a3a66645f06fd20c9061ff7e621f72b))
* discuss phase wihout modales ([6927293](https://github.com/jycamier/retrotro/commit/6927293872e3962fa55532ab43f996463b2d497a))
* initial commit ([d8f048e](https://github.com/jycamier/retrotro/commit/d8f048e614581b05b300892a4e056a28003268e9))
* multi-vote with visual tokens per participant ([0aa2dd0](https://github.com/jycamier/retrotro/commit/0aa2dd0adfb63c419074d6b7b01b4ec81763ebe7))
* replace PGBridge with MessageBus interface ([#11](https://github.com/jycamier/retrotro/issues/11)) ([e3656fb](https://github.com/jycamier/retrotro/commit/e3656fb7b850521858e06344cbf7f3c432fd7a38))


### Bug Fixes

* **deps:** upgrade dependencies to address security vulnerabilities ([8bca84d](https://github.com/jycamier/retrotro/commit/8bca84da37b619b3430de2738037a25b48c668a2))
* re-grouping items loses previously grouped children ([d0ccf83](https://github.com/jycamier/retrotro/commit/d0ccf8390c3549f5075799e312c56b81902bd6d3))
* reconnection-latency tests - adjust selectors and timeouts for page reload scenarios ([2d96ff3](https://github.com/jycamier/retrotro/commit/2d96ff3e2ff3d074753b32638d0cc28569edf733))
* release please configuration ([54bc05d](https://github.com/jycamier/retrotro/commit/54bc05d0a5faa1b34c3c7ff3a188a7d8d71c7020))
* set nats jetstream to disabled ([a642e59](https://github.com/jycamier/retrotro/commit/a642e593ab422e8d4e0bc7a33c13114e8a5e4fae))
* **websocket:** add reconnection handling with disconnect grace period ([#9](https://github.com/jycamier/retrotro/issues/9)) ([a158183](https://github.com/jycamier/retrotro/commit/a1581835df361faffb494045d85b9a262ab6985d))

## [0.0.3](https://github.com/jycamier/retrotro/compare/retrotro-backend-v0.0.2...retrotro-backend-v0.0.3) (2026-02-25)


### Features

* add detailed logging for item creation and WebSocket broadcasting ([41d5d45](https://github.com/jycamier/retrotro/commit/41d5d45523d57f019a260676f0f6c32996bb870a))
* add drag & drop and assignee management to team actions Kanban ([0140af3](https://github.com/jycamier/retrotro/commit/0140af305f3beb703ef0177fe08faadfc0ac9baf))
* add nats secrets management ([0afcce3](https://github.com/jycamier/retrotro/commit/0afcce3a578c6d4535d9effe32ead93ddd8a4428))
* add pgbridge for events ([b90eebf](https://github.com/jycamier/retrotro/commit/b90eebfc0c4a1828c3b7f5487984f08c0af2672c))
* add team actions Kanban view ([2c3d3c2](https://github.com/jycamier/retrotro/commit/2c3d3c221a3a66645f06fd20c9061ff7e621f72b))
* discuss phase wihout modales ([6927293](https://github.com/jycamier/retrotro/commit/6927293872e3962fa55532ab43f996463b2d497a))
* multi-vote with visual tokens per participant ([0aa2dd0](https://github.com/jycamier/retrotro/commit/0aa2dd0adfb63c419074d6b7b01b4ec81763ebe7))
* replace PGBridge with MessageBus interface ([#11](https://github.com/jycamier/retrotro/issues/11)) ([e3656fb](https://github.com/jycamier/retrotro/commit/e3656fb7b850521858e06344cbf7f3c432fd7a38))


### Bug Fixes

* re-grouping items loses previously grouped children ([d0ccf83](https://github.com/jycamier/retrotro/commit/d0ccf8390c3549f5075799e312c56b81902bd6d3))
* reconnection-latency tests - adjust selectors and timeouts for page reload scenarios ([2d96ff3](https://github.com/jycamier/retrotro/commit/2d96ff3e2ff3d074753b32638d0cc28569edf733))
* **websocket:** add reconnection handling with disconnect grace period ([#9](https://github.com/jycamier/retrotro/issues/9)) ([a158183](https://github.com/jycamier/retrotro/commit/a1581835df361faffb494045d85b9a262ab6985d))

## [0.0.2](https://github.com/jycamier/retrotro/compare/retrotro-backend-v0.0.1...retrotro-backend-v0.0.2) (2026-01-24)


### Features

* initial commit ([d8f048e](https://github.com/jycamier/retrotro/commit/d8f048e614581b05b300892a4e056a28003268e9))


### Bug Fixes

* **deps:** upgrade dependencies to address security vulnerabilities ([8bca84d](https://github.com/jycamier/retrotro/commit/8bca84da37b619b3430de2738037a25b48c668a2))
* release please configuration ([54bc05d](https://github.com/jycamier/retrotro/commit/54bc05d0a5faa1b34c3c7ff3a188a7d8d71c7020))
