---
name: package-docs
description: Write package documentation following the established patterns in this project. Use when asked to write docs, create documentation, or document a package.
argument-hint: [package-name]
---

## Required Process

1. Read the package source code first to understand all types, interfaces, and functions
2. Determine important public APIs. Ask user for clarification of what is important
3. Check for existing demo app in `demo-app/cmd/` that uses this package
4. Create the MDX file in `docs/src/content/docs/packages/`

## Required Document Structure

Every package doc must include these sections in this exact order:

1. Frontmatter - YAML with `title: packagename`
2. Imports - Starlight components: `LinkButton`, `Steps`, optionally `Code`
3. Introduction - Single sentence describing what the package provides
4. Core Components - Bulleted list of main types/interfaces with one-line descriptions
6. Link to pkg.go.dev in format `[Full package docs at pkg.go.dev](https://pkg.go.dev/github.com/platforma-dev/platforma/<package-name>)` 
5. Step-by-step guide - Using `<Steps>` component with 4-7 numbered steps
6. Using with Application - How to integrate with the `application` package
7. Complete example - Import from `demo-app/cmd/` or provide inline

## Writing Guidelines

- Single sentence introductions: Start with "The `packagename` package provides..."
- Note Runner interface: If a type implements `Runner`, mention "Implements `Runner` interface so it can be used as an `application` service."
- Include expected output: Show terminal/log output in step-by-step guides to help verify correctness
- Show integration pattern: Always demonstrate `app.RegisterService()` in "Using with Application" section
- Prefer demo imports: Use existing demo apps over inline code for complete examples

## MDX Formatting

Code blocks inside `<Steps>` require 4-space indentation:

```mdx
<Steps>

1. First step description

    ```go
    code := "indented with 4 spaces"
    ```

    Explanation paragraph also indented 4 spaces.

2. Second step...

</Steps>
```

## File Locations

- Package docs: `docs/src/content/docs/packages/{packagename}.mdx`
- Demo apps: `demo-app/cmd/{packagename}/main.go`
- Package source: `{packagename}/` or `internal/{packagename}/`

## Available Starlight Components

```mdx
import { LinkButton, Steps } from '@astrojs/starlight/components';
import { Code } from '@astrojs/starlight/components';
```

For importing demo code:
```mdx
import importedCode from '../../../../../demo-app/cmd/{package}/main.go?raw';

<Code code={importedCode} lang="go" title="{package}.go" />
```

## Template

```mdx
---
title: packagename
---
import { LinkButton, Steps } from '@astrojs/starlight/components';

The `packagename` package provides [single sentence description].

Core Components:

- `MainType`: Description. Implements `Runner` interface so it can be used as an `application` service.
- `Interface`: Description
- `HelperFunc`: Description

## Step-by-step guide

<Steps>

1. Step description

    ```go
    code here
    ```

    Explanation.

2. Next step...

</Steps>

## Using with Application

Since `MainType` implements the `Runner` interface, it can be registered as a service in an `Application`:

```go
app := application.New()

// setup code
app.RegisterService("service-name", instance)

app.Run(ctx)
```

## Complete example

import { Code } from '@astrojs/starlight/components';
import importedCode from '../../../../../demo-app/cmd/packagename/main.go?raw';

<Code code={importedCode} lang="go" title="packagename.go" />
```
