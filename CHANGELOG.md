# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Types of changes

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Removed` for now removed features.
- `Fixed` for any bug fixes.
- `Security` in case of vulnerabilities.

## [0.5.0] 2022-11-30

- `Added` Rate metrics
- `Added` Producers back pressure detection
- `Added` Optionaly wait for configured consumers and producers pool

## [0.4.0] 2022-07-05

- `Added` Cat Pipe can execute process and capture stdout or stderr stream

## [0.3.0] 2022-06-14

- `Added` Cat Pipe retry to connect to a failed tcp stream every second at startup.
- `Fixed` Fix blocking state when producer close before consumer connect.

## [0.2.0] 2022-05-23

- `Added` Cat Pipe client.

## [0.1.0] 2021-09-24

- `Added` First official release.
