param(
    [string]$Document = (Join-Path $PSScriptRoot '..\docs\代码实现与理论核验.md'),
    [string]$OutputDirectory = (Join-Path $PSScriptRoot '..\.bin\reference-from-document')
)

$ErrorActionPreference = 'Stop'
$workspace = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$allowedRoot = [System.IO.Path]::GetFullPath((Join-Path $workspace '.bin'))
$outputRoot = [System.IO.Path]::GetFullPath($OutputDirectory)
$separator = [System.IO.Path]::DirectorySeparatorChar
$prefix = $outputRoot.TrimEnd($separator) + $separator
if (-not $outputRoot.StartsWith($allowedRoot.TrimEnd($separator) + $separator, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw 'OutputDirectory 必须是本仓库 .bin 下的独立子目录。'
}
if (Test-Path -LiteralPath $outputRoot) {
    if (-not (Get-Item -LiteralPath $outputRoot).PSIsContainer) { throw '输出路径不是目录。' }
    if (@(Get-ChildItem -LiteralPath $outputRoot -Force).Count -ne 0) {
        throw '输出目录非空。请指定新的 .bin 子目录，以保留之前的实现。'
    }
}

$text = [System.IO.File]::ReadAllText([System.IO.Path]::GetFullPath($Document))
$text = $text.Replace("`r`n", "`n")
$pattern = '(?m)^<!-- file: (?<path>[^>\r\n]+) -->\n<!-- sha256: (?<hash>[0-9a-f]{64}) -->\n(?<fence>`{3,})[^\r\n]*\n(?<body>[\s\S]*?)^\k<fence>[ \t]*$'
$blocks = [regex]::Matches($text, $pattern)
if ($blocks.Count -eq 0) { throw '没有找到带校验信息的完整文件代码块。' }
$utf8 = New-Object System.Text.UTF8Encoding($false)
$sha = [System.Security.Cryptography.SHA256]::Create()
$files = New-Object 'System.Collections.Generic.Dictionary[string,byte[]]' ([System.StringComparer]::OrdinalIgnoreCase)
try {
    foreach ($block in $blocks) {
        $relative = $block.Groups['path'].Value
        if ([System.IO.Path]::IsPathRooted($relative) -or $relative -match '(^|[\/])\.\.([\/]|$)' -or $relative.Contains(':')) {
            throw "无效文件路径：$relative"
        }
        $target = [System.IO.Path]::GetFullPath((Join-Path $outputRoot $relative))
        if (-not $target.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) { throw "路径越界：$relative" }
        if ($files.ContainsKey($target)) { throw "文件重复：$relative" }
        $bytes = $utf8.GetBytes($block.Groups['body'].Value)
        $actualHash = [System.BitConverter]::ToString($sha.ComputeHash($bytes)).Replace('-', '').ToLowerInvariant()
        if ($actualHash -ne $block.Groups['hash'].Value) { throw "代码块校验失败：$relative" }
        $files.Add($target, $bytes)
    }
} finally { $sha.Dispose() }

# Validate every path and checksum before writing any file.
New-Item -ItemType Directory -Force -Path $outputRoot | Out-Null
foreach ($entry in $files.GetEnumerator()) {
    New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($entry.Key)) | Out-Null
    [System.IO.File]::WriteAllBytes($entry.Key, $entry.Value)
}
Write-Output "已提取 $($files.Count) 个文件至 $outputRoot"
Write-Output '这是独立参考工程；主项目源码未被覆盖。'
