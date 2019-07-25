import { unrecognizedValue } from 'f61ui/utils';
import { File } from 'generated/stoserver_types';

export enum Filetype {
	Word,
	Excel,
	Powerpoint,
	Spreadsheet,
	Picture,
	Video,
	Pdf,
	Audio,
	Archive,
	Other,
}

export function iconForFiletype(cl: Filetype) {
	switch (cl) {
		case Filetype.Word:
			return 'word.png';
		case Filetype.Excel:
			return 'excel.png';
		case Filetype.Powerpoint:
			return 'powerpoint.png';
		case Filetype.Spreadsheet:
			return 'spreadsheet.png';
		case Filetype.Picture:
			return 'picture.png';
		case Filetype.Video:
			return 'video.png';
		case Filetype.Pdf:
			return 'pdf.png';
		case Filetype.Audio:
			return 'audio.png';
		case Filetype.Archive:
			return 'archive.png';
		case Filetype.Other:
			return 'generic.png';
		default:
			throw unrecognizedValue(cl);
	}
}

export function filetypeForFile(file: File): Filetype {
	const ext = /\.([^.]+)$/.exec(file.Path);
	if (!ext) {
		return Filetype.Other;
	}

	switch (ext[1].toLowerCase()) {
		case 'doc':
		case 'docx':
			return Filetype.Word;
		case 'xls':
		case 'xlsx':
			return Filetype.Excel;
		case 'ppt':
		case 'pptx':
			return Filetype.Powerpoint;
		case 'ods':
			return Filetype.Spreadsheet;
		case 'jpg':
		case 'jpeg':
		case 'gif':
		case 'png':
		case 'bmp':
			return Filetype.Picture;
		case 'avi':
		case 'divx':
		case 'mp4':
		case 'mkv':
		case 'mov':
		case 'flv':
		case 'wmv':
		case '3gp':
		case 'webm':
			return Filetype.Video;
		case 'pdf':
			return Filetype.Pdf;
		case 'mp3':
		case 'wav':
		case 'mid':
		case 'flac':
		case 'ogg':
			return Filetype.Audio;
		case 'zip':
		case 'gz':
		case 'tar':
		case '7z':
			return Filetype.Archive;
		default:
			return Filetype.Other;
	}
}
