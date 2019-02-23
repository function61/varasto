// https://twitter.com/joonas_fi/status/1098572404975169536

interface Mag {
	suff: string;
	lt: number;
}

const mags: Mag[] = [
	{ suff: 'B', lt: 1024 },
	{ suff: 'kB', lt: 1024 * 1024 },
	{ suff: 'MB', lt: 1024 * 1024 * 1024 },
	{ suff: 'GB', lt: 1024 * 1024 * 1024 * 1024 },
	{ suff: 'TB', lt: 1024 * 1024 * 1024 * 1024 * 1024 },
	{ suff: 'PB', lt: 1024 * 1024 * 1024 * 1024 * 1024 * 1024 },
];

export function bytesToHumanReadable(num: number): string {
	let mag = mags[0];

	for (let i = 0; num >= mag.lt && i < mags.length; i++) {
		mag = mags[i];
	}

	const numDivided = num / (mag.lt / 1024);

	return numDivided.toFixed(2) + ' ' + mag.suff;
}
