import { Panel } from 'f61ui/component/bootstrap';
import { globalConfig } from 'f61ui/globalconfig';
import { jsxChildType } from 'f61ui/types';
import {
	MetadataBackdrop,
	MetadataHomepage,
	MetadataImdbId,
	MetadataKv,
	MetadataOverview,
	MetadataReleaseDate,
	MetadataTheMovieDbMovieId,
	MetadataTheMovieDbTvId,
	MetadataTheTvDbSeriesId,
	MetadataThumbnail,
	MetadataVideoRevenueDollars,
	MetadataVideoRuntimeMins,
} from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';

interface MetadataKeyValue {
	[key: string]: string;
}

interface MetadataPanelProps {
	data: MetadataKeyValue;
}

export class MetadataPanel extends React.Component<MetadataPanelProps, {}> {
	render() {
		const metadata = this.props.data;

		if (Object.keys(metadata).length === 0) {
			return '';
		}

		const backdropImage =
			metadata[MetadataBackdrop] ||
			metadata[MetadataThumbnail] ||
			globalConfig().assetsDir + '/../image-not-available.png';

		const overview: string = metadata[MetadataOverview] || '';

		const badges: jsxChildType[] = [];

		if (MetadataVideoRuntimeMins in metadata) {
			let hours = +metadata[MetadataVideoRuntimeMins] / 60;
			const minutes = Math.round(60 * (hours % 1));
			hours = Math.round(hours - (hours % 1));

			badges.push(
				<span className="label label-default margin-left">
					🕒 {hours}h {minutes}m
				</span>,
			);
		}

		if (MetadataVideoRevenueDollars in metadata) {
			badges.push(
				<span className="label label-default margin-left">
					💵 {Math.round(+metadata[MetadataVideoRevenueDollars] / 10000) / 100} million
				</span>,
			);
		}

		if (MetadataReleaseDate in metadata) {
			badges.push(
				<span className="label label-default margin-left">
					📅 {metadata[MetadataReleaseDate]}
				</span>,
			);
		}

		return (
			<Panel
				children={<img src={backdropImage} style={{ maxWidth: '100%' }} />}
				footer={
					<div>
						{overview}
						&nbsp;
						{badges}
						{this.maybeUrl(
							'thetvdb.com',
							'https://www.thetvdb.com/dereferrer/series/{key}',
							metadata[MetadataTheTvDbSeriesId],
						)}
						{this.maybeUrl(
							'IMDB',
							'https://www.imdb.com/title/{key}/',
							metadata[MetadataImdbId],
						)}
						{this.maybeUrl(
							'TMDb',
							'https://www.themoviedb.org/movie/{key}',
							metadata[MetadataTheMovieDbMovieId],
						)}
						{this.maybeUrl(
							'TMDb',
							'https://www.themoviedb.org/tv/{key}',
							metadata[MetadataTheMovieDbTvId],
						)}
						{this.maybeUrl('Homepage', '{key}', metadata[MetadataHomepage])}
					</div>
				}
			/>
		);
	}

	private maybeUrl(label: string, template: string, key?: string): jsxChildType {
		if (!key) {
			return '';
		}

		const url = template.replace('{key}', key);

		return (
			<a href={url} target="_blank">
				<span className="label label-default margin-left">🔗 {label}</span>
			</a>
		);
	}
}

export function metadataKvsToKv(kvs: MetadataKv[]): MetadataKeyValue {
	const ret: MetadataKeyValue = {};
	kvs.forEach((kv) => {
		ret[kv.Key] = kv.Value;
	});
	return ret;
}
