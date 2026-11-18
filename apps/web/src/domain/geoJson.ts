import type { Coordinate } from './coordinate'
import type { Report } from './report'

export interface GeoJsonFeature {
  readonly type: 'Feature'
  readonly geometry: { readonly type: 'Polygon'; readonly coordinates: number[][][] }
  readonly properties: Readonly<Record<string, string | number>>
}

export function reportToGeoJson(report: Report, points: readonly Coordinate[]): GeoJsonFeature {
  const ring = points.map((point) => [point.longitude, point.latitude])
  const first = ring[0]
  if (first !== undefined) {
    ring.push([...first])
  }

  return {
    type: 'Feature',
    geometry: { type: 'Polygon', coordinates: [ring] },
    properties: {
      report_id: report.id,
      plot_id: report.plotId,
      name: report.plotName,
      plot_version: report.plotVersion,
      area_m2: report.measurement.area.squareMeters,
      area_hectares: report.measurement.area.hectares,
      perimeter_m: report.measurement.perimeterMeters,
      created_at: report.createdAt.toISOString(),
    },
  }
}
