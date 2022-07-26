## Images
Build: `{{- .BuildImage -}}`
Run: `{{- .RunImage -}}`

## Patched USNs
{{- if ne (len .PatchedArray) 0 }}
{{ range .PatchedArray }}
- [{{- .Title -}}]({{- .URL -}})
{{- end }}
{{- else }}
No USNs patched in this release.
{{- end }}

## Build Image Package Changes
### Added
{{- if ne (len .BuildAdded) 0 }}
```
{{- range .BuildAdded }}
{{ .Name }} {{ .Version }} (PURL: {{ .PURL }})
{{- end }}
```
{{- else }}
No packages added.
{{- end }}

### Modified
{{- if ne (len .BuildModified) 0 }}
```
{{- range .BuildModified }}
{{ .Name }} {{ .PreviousVersion }} ==> {{ .CurrentVersion }} (PURL: {{ .PreviousPURL }} ==> {{ .CurrentPURL }})
{{- end }}
```
{{- else }}
No packages modified.
{{- end }}

## Run Image Package Changes
### Added
{{- if ne (len .RunAdded) 0 }}
```
{{- range .RunAdded }}
{{ .Name }} {{ .Version }} (PURL: {{ .PURL }})
{{- end }}
```
{{- else }}
No packages added.
{{- end }}

### Modified
{{- if ne (len .RunModified) 0 }}
```
{{- range .RunModified }}
{{ .Name }} {{ .PreviousVersion }} ==> {{ .CurrentVersion }} (PURL: {{ .PreviousPURL }} ==> {{ .CurrentPURL }})
{{- end }}
```
{{- else }}
No packages modified.
{{- end }}
