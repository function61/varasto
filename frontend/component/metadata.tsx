import { CollectionTagView } from 'component/tags';
import { DefaultLabel, Panel } from 'f61ui/component/bootstrap';
import { globalConfig } from 'f61ui/globalconfig';
import { collectionUrl } from 'generated/frontend_uiroutes';
import {
	downloadFileUrl,
	igdbIntegrationRedirUrl,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	BannerPath,
	CollectionSubsetWithMeta,
	MetadataAppleAppStoreApp,
	MetadataGogSlug,
	MetadataGooglePlayApp,
	MetadataHomepage,
	MetadataIgdbGameId,
	MetadataImdbId,
	MetadataKv,
	MetadataOverview,
	MetadataRedditSlug,
	MetadataReleaseDate,
	MetadataSteamAppId,
	MetadataTheMovieDbMovieId,
	MetadataTheMovieDbTvId,
	MetadataTitle,
	MetadataVideoRevenueDollars,
	MetadataVideoRuntimeMins,
	MetadataWikipediaSlug,
	MetadataYoutubeId,
} from 'generated/stoserver/stoservertypes_types';
import * as React from 'react';
interface MetadataKeyValue {
	[key: string]: string;
}

interface MetadataPanelProps {
	showTitle?: boolean; // shows title and publication date
	showDetails?: boolean; // shows summary, publication date and external links
	imageLinksToCollection?: boolean;
	collWithMeta: CollectionSubsetWithMeta;
}

export class MetadataPanel extends React.Component<MetadataPanelProps, {}> {
	render() {
		// shorthands
		const collWithMeta = this.props.collWithMeta;
		const coll = collWithMeta.Collection;

		const metadata = metadataKvsToKv(coll.Metadata);

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

		const bannerSrc =
			collWithMeta.FilesInMeta.indexOf(BannerPath) !== -1
				? downloadFileUrl(coll.Id, collWithMeta.FilesInMetaAt, BannerPath)
				: imageNotAvailable();

		const title = metadata[MetadataTitle] || '';

		const banner = <img src={bannerSrc} style={{ minWidth: '100%', maxWidth: '100%' }} />;

		const bannerMaybeLink = this.props.imageLinksToCollection ? (
			<a
				href={collectionUrl({
					id: coll.Id,
				})}>
				{banner}
			</a>
		) : (
			banner
		);

		return (
			<Panel
				heading={
					this.props.showTitle && (
						<div>
							{coll.Name} - {title} &nbsp;
							{badges}
							<CollectionTagView collection={coll} />
						</div>
					)
				}
				footer={
					this.props.showDetails && (
						<div>
							{title && <b>{title}.&nbsp;</b>}
							{overview}
							&nbsp;
							{badges.map((badge) => (
								<span className="margin-left">{badge}</span>
							))}
							{this.externalLinks(metadata)}
						</div>
					)
				}
				bodyMarginless={true}>
				{bannerMaybeLink}
			</Panel>
		);
	}

	private externalLinks(metadata: MetadataKeyValue): React.ReactNode {
		const links: Array<[string, string, string | undefined]> = [
			['IMDB', 'https://www.imdb.com/title/{key}/', metadata[MetadataImdbId]],
			['TMDb', 'https://www.themoviedb.org/movie/{key}', metadata[MetadataTheMovieDbMovieId]],
			['TMDb', 'https://www.themoviedb.org/tv/{key}', metadata[MetadataTheMovieDbTvId]],
			[
				'IGDB',
				'{key}',
				metadata[MetadataIgdbGameId] &&
					igdbIntegrationRedirUrl(metadata[MetadataIgdbGameId]),
			],
			[
				'Steam store',
				'https://store.steampowered.com/app/{key}',
				metadata[MetadataSteamAppId],
			],
			['GOG.com', 'https://www.gog.com/game/{key}', metadata[MetadataGogSlug]],
			['Wikipedia', 'https://en.wikipedia.org/wiki/{key}', metadata[MetadataWikipediaSlug]],
			['YouTube', 'https://www.youtube.com/watch?v={key}', metadata[MetadataYoutubeId]],
			['Reddit', 'https://www.reddit.com/r/{key}/', metadata[MetadataRedditSlug]],
			[
				'Google Play',
				'https://play.google.com/store/apps/details?id={key}',
				metadata[MetadataGooglePlayApp],
			],
			[
				'Apple App Store',
				'https://apps.apple.com/us/app/redirect/id{key}',
				metadata[MetadataAppleAppStoreApp],
			],
			['Homepage', '{key}', metadata[MetadataHomepage]],
		];

		return (
			<span>
				{links.map((linkDef) => {
					const [label, template, key] = linkDef;

					if (!key) {
						return null;
					}

					const url = template.replace('{key}', key);

					return (
						<a href={url} target="_blank" className="margin-left">
							<DefaultLabel>🔗 {label}</DefaultLabel>
						</a>
					);
				})}
			</span>
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

export function imageNotAvailable(): string {
	return globalConfig().assetsDir + '/../image-not-available.svg';
}
