#requires -version 5.1
<#
  Generate the llama.cpp Manager app icon (dark tech style + llama silhouette).
  Outputs a 1024x1024 PNG; Wails generates icon.ico from it at build time.
  Usage: powershell -ExecutionPolicy Bypass -File tools\genicon.ps1
#>
param(
    [string]$Out = (Join-Path (Split-Path -Parent $PSScriptRoot) "build\appicon.png"),
    [int]$Size = 1024
)

Add-Type -AssemblyName System.Drawing

$script:scale = $Size / 1024.0

function S([double]$v) { return [float]($v * $script:scale) }

function P([double]$x, [double]$y) {
    $lx = 232 + $x * 2.8
    $ly = 240 + $y * 2.8
    return [System.Drawing.PointF]::new((S $lx), (S $ly))
}

$bmp = [System.Drawing.Bitmap]::new($Size, $Size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
$g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
$g.Clear([System.Drawing.Color]::Transparent)

# ---------- rounded-square background ----------
$margin = 72.0
$radius = 214.0
$rect = [System.Drawing.RectangleF]::new((S $margin), (S $margin), (S (1024 - 2 * $margin)), (S (1024 - 2 * $margin)))
$d = S ($radius * 2)

$bg = [System.Drawing.Drawing2D.GraphicsPath]::new()
$bg.AddArc($rect.X, $rect.Y, $d, $d, 180, 90)
$bg.AddArc($rect.Right - $d, $rect.Y, $d, $d, 270, 90)
$bg.AddArc($rect.Right - $d, $rect.Bottom - $d, $d, $d, 0, 90)
$bg.AddArc($rect.X, $rect.Bottom - $d, $d, $d, 90, 90)
$bg.CloseFigure()

$bgBrush = [System.Drawing.Drawing2D.LinearGradientBrush]::new(
    $rect,
    [System.Drawing.Color]::FromArgb(255, 34, 49, 74),
    [System.Drawing.Color]::FromArgb(255, 10, 17, 28),
    90.0)
$g.FillPath($bgBrush, $bg)

# top highlight (fade out over the top 42% of the tile)
$glowBrush = [System.Drawing.Drawing2D.LinearGradientBrush]::new(
    $rect,
    [System.Drawing.Color]::FromArgb(60, 79, 157, 255),
    [System.Drawing.Color]::FromArgb(0, 79, 157, 255),
    90.0)
$blend = [System.Drawing.Drawing2D.ColorBlend]::new(3)
$blend.Colors = @(
    [System.Drawing.Color]::FromArgb(60, 79, 157, 255),
    [System.Drawing.Color]::FromArgb(0, 79, 157, 255),
    [System.Drawing.Color]::FromArgb(0, 79, 157, 255))
$blend.Positions = @([float]0.0, [float]0.42, [float]1.0)
$glowBrush.InterpolationColors = $blend
$g.FillPath($glowBrush, $bg)

# thin border
$borderColor = [System.Drawing.Color]::FromArgb(90, 90, 150, 220)
$borderWidth = S 4
$borderPen = New-Object System.Drawing.Pen($borderColor, $borderWidth)
$g.DrawPath($borderPen, $bg)

# ---------- llama silhouette ----------
$llama = [System.Drawing.Drawing2D.GraphicsPath]::new()
$llama.StartFigure()
$llama.AddLine((P 96 186), (P 92 120))
$llama.AddBezier((P 92 120), (P 90 104), (P 90 92), (P 96 82))
$llama.AddBezier((P 96 82), (P 70 74), (P 64 60), (P 72 48))
$llama.AddBezier((P 72 48), (P 74 30), (P 80 20), (P 86 18))
$llama.AddBezier((P 86 18), (P 96 24), (P 100 38), (P 100 50))
$llama.AddBezier((P 100 50), (P 106 30), (P 114 18), (P 122 20))
$llama.AddBezier((P 122 20), (P 132 26), (P 134 42), (P 130 54))
$llama.AddBezier((P 130 54), (P 140 58), (P 146 64), (P 148 74))
$llama.AddBezier((P 148 74), (P 150 84), (P 156 88), (P 164 90))
$llama.AddBezier((P 164 90), (P 180 92), (P 186 102), (P 184 114))
$llama.AddBezier((P 184 114), (P 182 124), (P 172 128), (P 160 128))
$llama.AddBezier((P 160 128), (P 146 128), (P 136 132), (P 130 142))
$llama.AddBezier((P 130 142), (P 124 150), (P 122 164), (P 124 186))
$llama.CloseFigure()

# outer glow
$glowSpecs = @(
    @(30, 26),
    @(44, 16),
    @(58, 8)
)
foreach ($spec in $glowSpecs) {
    $alpha = [int]$spec[0]
    $w = [double]$spec[1]
    $pen = New-Object System.Drawing.Pen([System.Drawing.Color]::FromArgb($alpha, 79, 157, 255), (S $w))
    $pen.LineJoin = [System.Drawing.Drawing2D.LineJoin]::Round
    $g.DrawPath($pen, $llama)
    $pen.Dispose()
}

# body gradient
$llamaBounds = $llama.GetBounds()
$llamaBrush = [System.Drawing.Drawing2D.LinearGradientBrush]::new(
    $llamaBounds,
    [System.Drawing.Color]::FromArgb(255, 79, 157, 255),
    [System.Drawing.Color]::FromArgb(255, 34, 211, 238),
    55.0)
$g.FillPath($llamaBrush, $llama)

# eye
$eye = [System.Drawing.RectangleF]::new((S (232 + 124 * 2.8)), (S (240 + 72 * 2.8)), (S 22), (S 22))
$eyeBrush = [System.Drawing.SolidBrush]::new([System.Drawing.Color]::FromArgb(255, 12, 20, 32))
$g.FillEllipse($eyeBrush, $eye)

# ---------- output ----------
$dir = Split-Path -Parent $Out
if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)

$g.Dispose()
$bmp.Dispose()

# Wails only generates icon.ico when it does not exist, so remove the stale one.
$ico = Join-Path (Split-Path -Parent $Out) "windows\icon.ico"
if (Test-Path $ico) {
    Remove-Item -LiteralPath $ico -Force
    Write-Host "removed stale $ico (wails will regenerate it on next build)"
}

Write-Host "icon written: $Out ($Size x $Size)"
