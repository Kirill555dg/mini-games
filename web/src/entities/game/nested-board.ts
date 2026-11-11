// Coordinate projections only; closures and the next board come from the server.
export function regionForCell(cell: number, width: number, size: number) {
  return Math.floor(Math.floor(cell / width) / size) * (width / size) + Math.floor((cell % width) / size);
}

export function cellsInRegion(region: number, width: number, size: number) {
  const columns = width / size;
  const row = Math.floor(region / columns) * size;
  const column = (region % columns) * size;
  return Array.from(
    { length: size * size },
    (_, cell) => (row + Math.floor(cell / size)) * width + column + (cell % size),
  );
}
