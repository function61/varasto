import { DefaultLabel, Panel } from 'f61ui/component/bootstrap';
import { globalConfig } from 'f61ui/globalconfig';
import {
	MetadataBackdrop,
	MetadataHomepage,
	MetadataImdbId,
	MetadataKv,
	MetadataOverview,
	MetadataReleaseDate,
	MetadataTheMovieDbMovieId,
	MetadataTheMovieDbTvId,
	MetadataIgdbGameId,
	MetadataTheTvDbSeriesId,
	MetadataThumbnail,
	MetadataGooglePlayApp,
	MetadataAppleAppStoreApp,
	MetadataSteamAppId,
	MetadataGogSlug,
	MetadataYoutubeId,
	MetadataRedditSlug,
	MetadataWikipediaSlug,
	MetadataVideoRevenueDollars,
	MetadataVideoRuntimeMins,
} from 'generated/stoserver/stoservertypes_types';
import { igdbIntegrationRedirUrl } from 'generated/stoserver/stoservertypes_endpoints';
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
			return null;
		}

		const overview: string = metadata[MetadataOverview] || '';

		const badges: React.ReactNode[] = [];

		if (MetadataVideoRuntimeMins in metadata) {
			let hours = +metadata[MetadataVideoRuntimeMins] / 60;
			const minutes = Math.round(60 * (hours % 1));
			hours = Math.round(hours - (hours % 1));

			badges.push(
				<DefaultLabel>
					🕒 {hours}h {minutes}m
				</DefaultLabel>,
			);
		}

		if (MetadataVideoRevenueDollars in metadata) {
			badges.push(
				<DefaultLabel>
					💵 {Math.round(+metadata[MetadataVideoRevenueDollars] / 10000) / 100} million
				</DefaultLabel>,
			);
		}

		if (MetadataReleaseDate in metadata) {
			badges.push(<DefaultLabel>📅 {metadata[MetadataReleaseDate]}</DefaultLabel>);
		}

		return (
			<Panel
				children={<img src={backdropImage(metadata)} style={{ maxWidth: '100%' }} />}
				footer={
					<div>
						{overview}
						&nbsp;
						{badges.map((badge) => (
							<span className="margin-left">{badge}</span>
						))}
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
						{this.maybeUrl(
							'IGDB',
							'{key}',
							metadata[MetadataIgdbGameId] &&
								igdbIntegrationRedirUrl(metadata[MetadataIgdbGameId]),
						)}
						{this.maybeUrl(
							'Steam store',
							'https://store.steampowered.com/app/{key}',
							metadata[MetadataSteamAppId],
						)}
						{this.maybeUrl(
							'GOG.com',
							'https://www.gog.com/game/{key}',
							metadata[MetadataGogSlug],
						)}
						{this.maybeUrl(
							'Wikipedia',
							'https://en.wikipedia.org/wiki/{key}',
							metadata[MetadataWikipediaSlug],
						)}
						{this.maybeUrl(
							'YouTube',
							'https://www.youtube.com/watch?v={key}',
							metadata[MetadataYoutubeId],
						)}
						{this.maybeUrl(
							'Reddit',
							'https://www.reddit.com/r/{key}/',
							metadata[MetadataRedditSlug],
						)}
						{this.maybeUrl(
							'Google Play',
							'https://play.google.com/store/apps/details?id={key}',
							metadata[MetadataGooglePlayApp],
						)}
						{this.maybeUrl(
							'Apple App Store',
							'https://apps.apple.com/us/app/redirect/id{key}',
							metadata[MetadataAppleAppStoreApp],
						)}
						{this.maybeUrl('Homepage', '{key}', metadata[MetadataHomepage])}
					</div>
				}
			/>
		);
	}

	private maybeUrl(label: string, template: string, key?: string): React.ReactNode {
		if (!key) {
			return null;
		}

		const url = template.replace('{key}', key);

		return (
			<a href={url} target="_blank" className="margin-left">
				<DefaultLabel>🔗 {label}</DefaultLabel>
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

export function backdropImage(metadata: MetadataKeyValue): string {
	return metadata[MetadataBackdrop] || metadata[MetadataThumbnail] || imageNotAvailable();
}

export function imageNotAvailable(): string {
	return globalConfig().assetsDir + '/../image-not-available.png';
}
