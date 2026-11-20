package document

import (
	_ "embed"
	"fmt"
	"math"
	"strconv"

	"github.com/signintech/gopdf"

	"github.com/vleshchenko/land-plot-system/services/report-service/internal/domain"
)

//go:embed fonts/Montserrat-Regular.ttf
var regularFont []byte

//go:embed fonts/Montserrat-Bold.ttf
var boldFont []byte

const (
	fontRegular     = "montserrat"
	fontBold        = "montserrat-bold"
	pageWidth       = 595.0
	pageHeight      = 842.0
	margin          = 48.0
	contentWidth    = pageWidth - 2*margin
	bottomLimit     = pageHeight - 64.0
	sizeTitle       = 20
	sizeSection     = 13
	sizeArea        = 18
	sizeBody        = 10
	sizeSmall       = 8.5
	lineBody        = 14.0
	lineSmall       = 12.0
	sectionGap      = 18.0
	boundaryHeight  = 210.0
	boundaryInset   = 14.0
	logoSize        = 18.0
	rowHeight       = 16.0
	compressLevel   = 6
	degreesHalfTurn = 180.0
	strokeOnly      = "D"
	fillAndStroke   = "DF"
	lineWidthThin   = 0.6
	lineWidthShape  = 1.6
	documentCreator = "Arqetra"
	half            = 2
	minimumShape    = 3
	sectionLine     = lineBody + 4
	headlineLine    = 24.0
	logoTextGap     = 8.0
	logoBottomGap   = 14.0
	rowRuleOffset   = 4.0
)

var logoCorners = [4][2]float64{{2, 3}, {logoSize - 1, 1}, {logoSize - 3, logoSize - 2}, {0, logoSize - 4}}

type color struct{ red, green, blue uint8 }

var (
	colorInk     = color{31, 42, 36}
	colorMuted   = color{91, 99, 94}
	colorPrimary = color{47, 107, 79}
	colorSoft    = color{234, 241, 236}
	colorBorder  = color{220, 219, 211}
	colorShape   = color{205, 224, 212}
)

var seasonColumnWidths = [7]float64{86, 66, 62, 62, 82, 70, 71}

var coordinateColumnWidths = [3]float64{60, 150, 150}

type Renderer struct{}

type writer struct {
	pdf   *gopdf.GoPdf
	text  texts
	y     float64
	err   error
	drawn []string
}

func (Renderer) PDF(report domain.Report, language domain.Language) ([]byte, error) {
	content, _, err := render(report, language)

	return content, err
}

func render(report domain.Report, language domain.Language) ([]byte, []string, error) {
	document := &writer{pdf: &gopdf.GoPdf{}, text: textsFor(language)}
	document.pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	document.pdf.SetCompressLevel(compressLevel)
	document.pdf.SetInfo(gopdf.PdfInfo{
		Title:        fmt.Sprintf(document.text.title, report.PlotName),
		Creator:      documentCreator,
		Producer:     documentCreator,
		CreationDate: report.CreatedAt,
	})

	for family, data := range map[string][]byte{fontRegular: regularFont, fontBold: boldFont} {
		if err := document.pdf.AddTTFFontData(family, data); err != nil {
			return nil, nil, fmt.Errorf(wrapAddFontFamily, family, err)
		}
	}

	document.newPage()
	document.header(report)
	document.boundary(report.Boundary)
	document.area(report)
	document.terrain(report.Terrain)
	document.climate(report.Climate)
	document.coordinates(report.Boundary)
	document.footer()

	if document.err != nil {
		return nil, nil, fmt.Errorf(wrapDrawPDF, document.err)
	}

	content, err := document.pdf.GetBytesPdfReturnErr()
	if err != nil {
		return nil, nil, fmt.Errorf(wrapOutputPDF, err)
	}

	return content, document.drawn, nil
}

func (document *writer) newPage() {
	document.pdf.AddPage()
	document.y = margin
}

func (document *writer) ensure(height float64) {
	if document.y+height > bottomLimit {
		document.newPage()
	}
}

func (document *writer) font(bold bool, size float64, ink color) {
	family := fontRegular
	if bold {
		family = fontBold
	}

	if err := document.pdf.SetFont(family, "", size); err != nil && document.err == nil {
		document.err = err
	}

	document.pdf.SetTextColor(ink.red, ink.green, ink.blue)
}

func (document *writer) write(x float64, value string) {
	document.drawn = append(document.drawn, value)
	document.pdf.SetXY(x, document.y)

	if err := document.pdf.Cell(nil, value); err != nil && document.err == nil {
		document.err = err
	}
}

func (document *writer) paragraph(value string, bold bool, size, lineHeight float64, ink color) {
	document.font(bold, size, ink)

	lines, err := document.pdf.SplitTextWithWordWrap(value, contentWidth)
	if err != nil {
		lines = []string{value}
	}

	for _, line := range lines {
		document.ensure(lineHeight)
		document.write(margin, line)
		document.y += lineHeight
	}
}

func (document *writer) section(title string) {
	document.y += sectionGap
	document.ensure(sectionGap * half)
	document.paragraph(title, true, sizeSection, sectionLine, colorInk)
}

func (document *writer) header(report domain.Report) {
	document.logo(margin, document.y)
	document.font(true, sizeSection, colorInk)
	document.write(margin+logoSize+logoTextGap, documentCreator)
	document.y += logoSize + logoBottomGap

	document.paragraph(fmt.Sprintf(document.text.title, report.PlotName), true, sizeTitle, headlineLine, colorInk)
	document.paragraph(
		fmt.Sprintf(document.text.meta, report.Number(), document.text.date(report.CreatedAt)),
		false, sizeBody, lineBody, colorMuted,
	)

	if report.PlotDeleted() {
		document.paragraph(document.text.plotDeleted, false, sizeBody, lineBody, colorMuted)
	}
}

func (document *writer) logo(x, y float64) {
	document.pdf.SetStrokeColor(colorPrimary.red, colorPrimary.green, colorPrimary.blue)
	document.pdf.SetLineWidth(lineWidthShape)

	corners := make([]gopdf.Point, len(logoCorners))
	for index, corner := range logoCorners {
		corners[index] = gopdf.Point{X: x + corner[0], Y: y + corner[1]}
	}

	document.pdf.Polygon(corners, strokeOnly)
}

func (document *writer) boundary(points []domain.Coordinate) {
	document.section(document.text.boundaryTitle)
	document.ensure(boundaryHeight)

	top := document.y
	document.pdf.SetFillColor(colorSoft.red, colorSoft.green, colorSoft.blue)
	document.pdf.SetStrokeColor(colorBorder.red, colorBorder.green, colorBorder.blue)
	document.pdf.SetLineWidth(lineWidthThin)
	document.pdf.RectFromUpperLeftWithStyle(margin, top, contentWidth, boundaryHeight, fillAndStroke)

	if len(points) < minimumShape {
		document.y = top + boundaryHeight/half
		document.paragraph(document.text.noData, false, sizeBody, lineBody, colorMuted)
		document.y = top + boundaryHeight

		return
	}

	shape := fitShape(
		points,
		margin+boundaryInset,
		top+boundaryInset,
		contentWidth-half*boundaryInset,
		boundaryHeight-half*boundaryInset,
	)

	document.pdf.SetFillColor(colorShape.red, colorShape.green, colorShape.blue)
	document.pdf.SetStrokeColor(colorPrimary.red, colorPrimary.green, colorPrimary.blue)
	document.pdf.SetLineWidth(lineWidthShape)
	document.pdf.Polygon(shape, fillAndStroke)
	document.y = top + boundaryHeight
}

func fitShape(points []domain.Coordinate, left, top, width, height float64) []gopdf.Point {
	centre := domain.Centroid(points)
	stretch := math.Cos(centre.Latitude * math.Pi / degreesHalfTurn)

	minimumX, maximumX := math.Inf(1), math.Inf(-1)
	minimumY, maximumY := math.Inf(1), math.Inf(-1)

	for _, point := range points {
		x, y := point.Longitude*stretch, -point.Latitude
		minimumX, maximumX = math.Min(minimumX, x), math.Max(maximumX, x)
		minimumY, maximumY = math.Min(minimumY, y), math.Max(maximumY, y)
	}

	scale := math.Min(width/math.Max(maximumX-minimumX, math.SmallestNonzeroFloat64),
		height/math.Max(maximumY-minimumY, math.SmallestNonzeroFloat64))
	offsetX := left + (width-(maximumX-minimumX)*scale)/half
	offsetY := top + (height-(maximumY-minimumY)*scale)/half

	shape := make([]gopdf.Point, len(points))
	for index, point := range points {
		shape[index] = gopdf.Point{
			X: offsetX + (point.Longitude*stretch-minimumX)*scale,
			Y: offsetY + (-point.Latitude-minimumY)*scale,
		}
	}

	return shape
}

func (document *writer) area(report domain.Report) {
	text := document.text
	document.section(text.areaTitle)

	headline := text.formatHectares(report.AreaHectares)
	if report.Accuracy != nil {
		headline += " " + text.formatMargin(*report.Accuracy)
	}

	document.paragraph(headline, true, sizeArea, headlineLine, colorInk)
	document.paragraph(fmt.Sprintf(text.areaDetails,
		text.number(report.AreaM2, 0), text.number(report.AreaAres, 1), text.formatMeters(report.PerimeterM),
	), false, sizeBody, lineBody, colorInk)

	if report.Accuracy == nil {
		document.paragraph(text.noAccuracy, false, sizeBody, lineBody, colorMuted)

		return
	}

	accuracy := report.Accuracy
	if accuracy.Basis == "assumed" {
		document.paragraph(text.approximateNote, false, sizeBody, lineBody, colorMuted)
	}

	document.paragraph(fmt.Sprintf(text.accuracyDetails,
		text.percentOf(accuracy.Confidence), text.percentOf(accuracy.ScaleShare),
		text.percentOf(accuracy.ObstructionShare), fmt.Sprintf(text.meters, text.number(accuracy.GroundShiftM, 0)),
	), false, sizeBody, lineBody, colorMuted)
}

func (document *writer) terrain(terrain *domain.Terrain) {
	text := document.text
	document.section(text.terrainTitle)

	if terrain == nil || terrain.Status != domain.DataAvailable {
		document.paragraph(text.noData, false, sizeBody, lineBody, colorMuted)

		return
	}

	document.paragraph(fmt.Sprintf(text.terrainDetails,
		text.number(terrain.SlopeDegrees, 1),
		text.formatHectares(terrain.SurfaceAreaM2/domain.SquareMetersPerHectare),
		text.percentValue(terrain.SurfaceIncreasePercent),
		text.number(terrain.MinElevationM, 0), text.formatMeters(terrain.MaxElevationM),
		text.formatMeters(terrain.ReliefM),
	), false, sizeBody, lineBody, colorInk)

	if terrain.Coarse {
		document.paragraph(text.coarse, false, sizeBody, lineBody, colorMuted)
	}

	document.paragraph(text.cadastral, false, sizeBody, lineBody, colorMuted)
}

func (document *writer) climate(climate *domain.Climate) {
	text := document.text

	if climate == nil || climate.Status != domain.DataAvailable {
		document.section(text.climateHeading)
		document.paragraph(text.noData, false, sizeBody, lineBody, colorMuted)

		return
	}

	document.section(fmt.Sprintf(text.climateTitle, climate.FirstYear, climate.LastYear))
	document.row(text.seasonColumns[:], seasonColumnWidths[:], true)

	for _, season := range climate.Seasons {
		document.row([]string{
			text.seasons[season.Season],
			text.formatCelsius(season.MeanC), text.formatCelsius(season.MeanMinC), text.formatCelsius(season.MeanMaxC),
			fmt.Sprintf(text.millimeters, text.number(season.PrecipitationMm, 0)),
			text.number(season.FrostDays, 0), text.number(season.HeatDays, 0),
		}, seasonColumnWidths[:], false)
	}

	document.y += lineSmall / half
	document.paragraph(
		fmt.Sprintf(text.lastSpringFrost, text.monthDay(climate.LastSpringFrost)),
		false,
		sizeBody,
		lineBody,
		colorInk,
	)
	document.paragraph(
		fmt.Sprintf(text.firstAutumnFrost, text.monthDay(climate.FirstAutumnFrost)),
		false,
		sizeBody,
		lineBody,
		colorInk,
	)
	document.paragraph(text.growing(climate.GrowingDays), false, sizeBody, lineBody, colorInk)
}

func (document *writer) row(cells []string, widths []float64, heading bool) {
	document.ensure(rowHeight)

	ink := colorInk
	if heading {
		ink = colorMuted
	}

	document.font(heading, sizeSmall, ink)

	x := margin
	for index, cell := range cells {
		document.write(x, cell)
		x += widths[index]
	}

	document.y += rowHeight
	document.pdf.SetStrokeColor(colorBorder.red, colorBorder.green, colorBorder.blue)
	document.pdf.SetLineWidth(lineWidthThin)
	document.pdf.Line(margin, document.y-rowRuleOffset, x, document.y-rowRuleOffset)
}

func (document *writer) coordinates(points []domain.Coordinate) {
	text := document.text
	document.section(text.coordinatesTitle)

	if len(points) == 0 {
		document.paragraph(text.noData, false, sizeBody, lineBody, colorMuted)

		return
	}

	document.row(text.coordinateColumns[:], coordinateColumnWidths[:], true)

	for index, point := range points {
		document.row([]string{
			strconv.Itoa(index + 1), text.coordinate(point.Latitude), text.coordinate(point.Longitude),
		}, coordinateColumnWidths[:], false)
	}
}

func (document *writer) footer() {
	document.y += sectionGap
	document.paragraph(document.text.sources, false, sizeSmall, lineSmall, colorMuted)
	document.paragraph(document.text.disclaimer, false, sizeSmall, lineSmall, colorMuted)
}
