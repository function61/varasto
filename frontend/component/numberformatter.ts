// thanks https://stackoverflow.com/a/2901298
export function thousandSeparate(x: number): string {
	return x.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
}
