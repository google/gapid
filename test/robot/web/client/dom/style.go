// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dom

import (
	"fmt"

	"github.com/gopherjs/gopherjs/js"
)

type CSSStyleDeclaration struct {
	*js.Object
	AlignContent              string        `js:"alignContent"`
	AlignItems                string        `js:"alignItems"`
	AlignSelf                 string        `js:"AlignSelf"`
	AlignmentBaseline         string        `js:"alignmentBaseline"`
	All                       string        `js:"all"`
	Animation                 string        `js:"animation"`
	AnimationDelay            string        `js:"animationDelay"`
	AnimationDirection        string        `js:"animationDirection"`
	AnimationDuration         string        `js:"animationDuration"`
	AnimationFillMode         string        `js:"animationFillMode"`
	AnimationIterationCount   string        `js:"animationIterationCount"`
	AnimationName             string        `js:"animationName"`
	AnimationPlayState        string        `js:"animationPlayState"`
	AnimationTimingFunction   string        `js:"animationTimingFunction"`
	BackfaceVisibility        string        `js:"backfaceVisibility"`
	Background                string        `js:"background"`
	BackgroundAttachment      string        `js:"backgroundAttachment"`
	BackgroundBlendMode       string        `js:"backgroundBlendMode"`
	BackgroundClip            string        `js:"backgroundClip"`
	BackgroundColor           Color         `js:"backgroundColor"`
	BackgroundImage           string        `js:"backgroundImage"`
	BackgroundOrigin          string        `js:"backgroundOrigin"`
	BackgroundPosition        string        `js:"backgroundPosition"`
	BackgroundPositionX       string        `js:"backgroundPositionX"`
	BackgroundPositionY       string        `js:"backgroundPositionY"`
	BackgroundRepeat          string        `js:"backgroundRepeat"`
	BackgroundRepeatX         string        `js:"backgroundRepeatX"`
	BackgroundRepeatY         string        `js:"backgroundRepeatY"`
	BackgroundSize            string        `js:"backgroundSize"`
	BaselineShift             string        `js:"baselineShift"`
	Border                    string        `js:"border"`
	BorderBottom              string        `js:"borderBottom"`
	BorderBottomColor         Color         `js:"borderBottomColor"`
	BorderBottomLeftRadius    string        `js:"borderBottomLeftRadius"`
	BorderBottomRightRadius   string        `js:"borderBottomRightRadius"`
	BorderBottomStyle         string        `js:"borderBottomStyle"`
	BorderBottomWidth         string        `js:"borderBottomWidth"`
	BorderCollapse            string        `js:"borderCollapse"`
	BorderColor               Color         `js:"borderColor"`
	BorderImage               string        `js:"borderImage"`
	BorderImageOutset         string        `js:"borderImageOutset"`
	BorderImageRepeat         string        `js:"borderImageRepeat"`
	BorderImageSlice          string        `js:"borderImageSlice"`
	BorderImageSource         string        `js:"borderImageSource"`
	BorderImageWidth          string        `js:"borderImageWidth"`
	BorderLeft                string        `js:"borderLeft"`
	BorderLeftColor           Color         `js:"borderLeftColor"`
	BorderLeftStyle           string        `js:"borderLeftStyle"`
	BorderLeftWidth           string        `js:"borderLeftWidth"`
	BorderRadius              string        `js:"borderRadius"`
	BorderRight               string        `js:"borderRight"`
	BorderRightColor          Color         `js:"borderRightColor"`
	BorderRightStyle          string        `js:"borderRightStyle"`
	BorderRightWidth          string        `js:"borderRightWidth"`
	BorderSpacing             string        `js:"borderSpacing"`
	BorderStyle               string        `js:"borderStyle"`
	BorderTop                 string        `js:"borderTop"`
	BorderTopColor            Color         `js:"borderTopColor"`
	BorderTopLeftRadius       string        `js:"borderTopLeftRadius"`
	BorderTopRightRadius      string        `js:"borderTopRightRadius"`
	BorderTopStyle            string        `js:"borderTopStyle"`
	BorderTopWidth            string        `js:"borderTopWidth"`
	BorderWidth               string        `js:"borderWidth"`
	Bottom                    string        `js:"bottom"`
	BoxShadow                 string        `js:"boxShadow"`
	BoxSizing                 string        `js:"boxSizing"`
	BreakAfter                string        `js:"breakAfter"`
	BreakBefore               string        `js:"breakBefore"`
	BreakInside               string        `js:"breakInside"`
	BufferedRendering         string        `js:"bufferedRendering"`
	CaptionSide               string        `js:"captionSide"`
	Clear                     string        `js:"clear"`
	Clip                      string        `js:"clip"`
	ClipPath                  string        `js:"clipPath"`
	ClipRule                  string        `js:"clipRule"`
	Color                     Color         `js:"color"`
	ColorInterpolation        string        `js:"colorInterpolation"`
	ColorInterpolationFilters string        `js:"colorInterpolationFilters"`
	ColorRendering            string        `js:"colorRendering"`
	ColumnCount               string        `js:"columnCount"`
	ColumnFill                string        `js:"columnFill"`
	ColumnGap                 string        `js:"columnGap"`
	ColumnRule                string        `js:"columnRule"`
	ColumnRuleColor           Color         `js:"columnRuleColor"`
	ColumnRuleStyle           string        `js:"columnRuleStyle"`
	ColumnRuleWidth           string        `js:"columnRuleWidth"`
	ColumnSpan                string        `js:"columnSpan"`
	ColumnWidth               string        `js:"columnWidth"`
	Columns                   string        `js:"columns"`
	Content                   string        `js:"content"`
	CounterIncrement          string        `js:"counterIncrement"`
	CounterReset              string        `js:"counterReset"`
	CSSFloat                  string        `js:"cssFloat"`
	CSSText                   string        `js:"cssText"`
	Cursor                    string        `js:"cursor"`
	Cx                        string        `js:"cx"`
	Cy                        string        `js:"cy"`
	Direction                 string        `js:"direction"`
	Display                   Display       `js:"display"`
	DominantBaseline          string        `js:"dominantBaseline"`
	EmptyCells                string        `js:"emptyCells"`
	Fill                      string        `js:"fill"`
	FillOpacity               string        `js:"fillOpacity"`
	FillRule                  string        `js:"fillRule"`
	Filter                    string        `js:"filter"`
	Flex                      string        `js:"flex"`
	FlexBasis                 string        `js:"flexBasis"`
	FlexDirection             FlexDirection `js:"flexDirection"`
	FlexFlow                  string        `js:"flexFlow"`
	FlexGrow                  string        `js:"flexGrow"`
	FlexShrink                string        `js:"flexShrink"`
	FlexWrap                  string        `js:"flexWrap"`
	Float                     string        `js:"float"`
	FloodColor                string        `js:"floodColor"`
	FloodOpacity              string        `js:"floodOpacity"`
	Font                      string        `js:"font"`
	FontFamily                string        `js:"fontFamily"`
	FontFeatureSettings       string        `js:"fontFeatureSettings"`
	FontKerning               string        `js:"fontKerning"`
	FontSize                  string        `js:"fontSize"`
	FontStretch               string        `js:"fontStretch"`
	FontStyle                 string        `js:"fontStyle"`
	FontVariant               string        `js:"fontVariant"`
	FontVariantLigatures      string        `js:"fontVariantLigatures"`
	FontWeight                string        `js:"fontWeight"`
	Height                    int           `js:"height"`
	ImageRendering            string        `js:"imageRendering"`
	Isolation                 string        `js:"isolation"`
	JustifyContent            string        `js:"justifyContent"`
	Left                      string        `js:"left"`
	Length                    string        `js:"length int"`
	LetterSpacing             string        `js:"letterSpacing"`
	LightingColor             Color         `js:"lightingColor"`
	LineHeight                string        `js:"lineHeight"`
	ListStyle                 string        `js:"listStyle"`
	ListStyleImage            string        `js:"listStyleImage"`
	ListStylePosition         string        `js:"listStylePosition"`
	ListStyleType             string        `js:"listStyleType"`
	Margin                    string        `js:"margin"`
	MarginBottom              string        `js:"marginBottom"`
	MarginLeft                string        `js:"marginLeft"`
	MarginRight               string        `js:"marginRight"`
	MarginTop                 string        `js:"marginTop"`
	Marker                    string        `js:"marker"`
	MarkerEnd                 string        `js:"markerEnd"`
	MarkerMid                 string        `js:"markerMid"`
	MarkerStart               string        `js:"markerStart"`
	Mask                      string        `js:"mask"`
	MaskType                  string        `js:"maskType"`
	MaxHeight                 int           `js:"maxHeight"`
	MaxWidth                  int           `js:"maxWidth"`
	MaxZoom                   string        `js:"maxZoom"`
	MinHeight                 int           `js:"minHeight"`
	MinWidth                  int           `js:"minWidth"`
	MinZoom                   string        `js:"minZoom"`
	MixBlendMode              string        `js:"mixBlendMode"`
	Motion                    string        `js:"motion"`
	MotionOffset              string        `js:"motionOffset"`
	MotionPath                string        `js:"motionPath"`
	MotionRotation            string        `js:"motionRotation"`
	ObjectFit                 string        `js:"objectFit"`
	ObjectPosition            string        `js:"objectPosition"`
	Opacity                   string        `js:"opacity"`
	Order                     string        `js:"order"`
	Orientation               string        `js:"orientation"`
	Orphans                   string        `js:"orphans"`
	Outline                   string        `js:"outline"`
	OutlineColor              Color         `js:"outlineColor"`
	OutlineOffset             string        `js:"outlineOffset"`
	OutlineStyle              string        `js:"outlineStyle"`
	OutlineWidth              string        `js:"outlineWidth"`
	Overflow                  string        `js:"overflow"`
	OverflowWrap              string        `js:"overflowWrap"`
	OverflowX                 string        `js:"overflowX"`
	OverflowY                 string        `js:"overflowY"`
	Padding                   string        `js:"padding"`
	PaddingBottom             string        `js:"paddingBottom"`
	PaddingLeft               string        `js:"paddingLeft"`
	PaddingRight              string        `js:"paddingRight"`
	PaddingTop                string        `js:"paddingTop"`
	Page                      string        `js:"page"`
	PageBreakAfter            string        `js:"pageBreakAfter"`
	PageBreakBefore           string        `js:"pageBreakBefore"`
	PageBreakInside           string        `js:"pageBreakInside"`
	PaintOrder                string        `js:"paintOrder"`
	ParentRule                string        `js:"parentRule"`
	Perspective               string        `js:"perspective"`
	PerspectiveOrigin         string        `js:"perspectiveOrigin"`
	PointerEvents             string        `js:"pointerEvents"`
	Position                  Position      `js:"position"`
	Quotes                    string        `js:"quotes"`
	R                         string        `js:"r"`
	Resize                    string        `js:"resize"`
	Right                     string        `js:"right"`
	Rx                        string        `js:"rx"`
	Ry                        string        `js:"ry"`
	ShapeImageThreshold       string        `js:"shapeImageThreshold"`
	ShapeMargin               string        `js:"shapeMargin"`
	ShapeOutside              string        `js:"shapeOutside"`
	ShapeRendering            string        `js:"shapeRendering"`
	Size                      string        `js:"size"`
	Speak                     string        `js:"speak"`
	Src                       string        `js:"src"`
	StopColor                 Color         `js:"stopColor"`
	StopOpacity               string        `js:"stopOpacity"`
	Stroke                    string        `js:"stroke"`
	StrokeDasharray           string        `js:"strokeDasharray"`
	StrokeDashoffset          string        `js:"strokeDashoffset"`
	StrokeLinecap             string        `js:"strokeLinecap"`
	StrokeLinejoin            string        `js:"strokeLinejoin"`
	StrokeMiterlimit          string        `js:"strokeMiterlimit"`
	StrokeOpacity             string        `js:"strokeOpacity"`
	StrokeWidth               string        `js:"strokeWidth"`
	TabSize                   string        `js:"tabSize"`
	TableLayout               string        `js:"tableLayout"`
	TextAlign                 string        `js:"textAlign"`
	TextAlignLast             string        `js:"textAlignLast"`
	TextAnchor                string        `js:"textAnchor"`
	TextCombineUpright        string        `js:"textCombineUpright"`
	TextDecoration            string        `js:"textDecoration"`
	TextIndent                string        `js:"textIndent"`
	TextOrientation           string        `js:"textOrientation"`
	TextOverflow              string        `js:"textOverflow"`
	TextRendering             string        `js:"textRendering"`
	TextShadow                string        `js:"textShadow"`
	TextTransform             string        `js:"textTransform"`
	Top                       string        `js:"top"`
	TouchAction               string        `js:"touchAction"`
	Transform                 string        `js:"transform"`
	TransformOrigin           string        `js:"transformOrigin"`
	TransformStyle            string        `js:"transformStyle"`
	Transition                string        `js:"transition"`
	TransitionDelay           string        `js:"transitionDelay"`
	TransitionDuration        string        `js:"transitionDuration"`
	TransitionProperty        string        `js:"transitionProperty"`
	TransitionTimingFunction  string        `js:"transitionTimingFunction"`
	UnicodeBidi               string        `js:"unicodeBidi"`
	UnicodeRange              string        `js:"unicodeRange"`
	UserZoom                  string        `js:"userZoom"`
	VectorEffect              string        `js:"vectorEffect"`
	VerticalAlign             string        `js:"verticalAlign"`
	Visibility                string        `js:"visibility"`
	WhiteSpace                string        `js:"whiteSpace"`
	Widows                    string        `js:"widows"`
	Width                     int           `js:"width"`
	WillChange                string        `js:"willChange"`
	WordBreak                 string        `js:"wordBreak"`
	WordSpacing               string        `js:"wordSpacing"`
	WordWrap                  string        `js:"wordWrap"`
	WritingMode               string        `js:"writingMode"`
	X                         string        `js:"x"`
	Y                         string        `js:"y"`
	ZIndex                    string        `js:"zIndex"`
	Zoom                      string        `js:"zoom"`
}

// Color is a CSS color.
type Color string

const (
	White       Color = "white"
	Black       Color = "black"
	Transparent Color = "transparent"
	Red         Color = "red"
	Green       Color = "green"
	Blue        Color = "blue"
	Yellow      Color = "yellow"
)

// Font is a font description.
type Font string

// NewFont returns a new font description with the specified size in pixels and
// family.
func NewFont(size int, family string) Font {
	return Font(fmt.Sprintf("%dpx %v", size, family))
}

// RGB returns a new Color with the specified red, green and blue values that
// each range from 0 to 1.
func RGB(r, g, b float64) Color {
	return Color(fmt.Sprint("rgb(", int(r*255), ",", int(g*255), ",", int(b*255), ")"))
}

// RGBA returns a new Color with the specified red, green, blue and alpha values
// that each range from 0 to 1.
func RGBA(r, g, b, a float64) Color {
	return Color(fmt.Sprint("rgba(", int(r*255), ",", int(g*255), ",", int(b*255), ",", a, ")"))
}

// Display is a CSS display enumerator.
type Display string

const (
	DisplayBlock            Display = "block"
	DisplayFlex             Display = "flex"
	DisplayGrid             Display = "grid"
	DisplayInherit          Display = "inherit"
	DisplayInitial          Display = "initial"
	DisplayInline           Display = "inline"
	DisplayInlineBlock      Display = "inline-block"
	DisplayInlineFlex       Display = "inline-flex"
	DisplayInlineGrid       Display = "inline-grid"
	DisplayInlineTable      Display = "inline-table"
	DisplayListItem         Display = "list-item"
	DisplayNone             Display = "list-none"
	DisplayRunIn            Display = "run-in"
	DisplayTable            Display = "table"
	DisplayTableCaption     Display = "table-caption"
	DisplayTableCell        Display = "table-cell"
	DisplayTableColumn      Display = "table-column"
	DisplayTableColumnGroup Display = "table-column-group"
	DisplayTableFooterGroup Display = "table-footer-group"
	DisplayTableHeaderGroup Display = "table-header-group"
	DisplayTableRow         Display = "table-row"
	DisplayTableRowGroup    Display = "table-row-group"
)

// FlexDirection is a CSS flexbox direction enumerator.
type FlexDirection string

const (
	FlexColumn        FlexDirection = "column"
	FlexColumnReverse FlexDirection = "column-reverse"
	FlexInherit       FlexDirection = "inherit"
	FlexInitial       FlexDirection = "initial"
	FlexRow           FlexDirection = "row"
	FlexRowReverse    FlexDirection = "row-reverse"
)

// Position is a CSS position enumerator.
type Position string

const (
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
	PositionInherit  Position = "inherit"
	PositionInitial  Position = "initial"
	PositionRelative Position = "relative"
	PositionStatic   Position = "static"
)
