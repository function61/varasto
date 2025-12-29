import { replicationPolicyAutocomplete, tmdbTvAutocomplete } from 'component/autocompletes';
import { collectionDropdown } from 'component/collectiondropdown';
import { DocLink } from 'component/doclink';
import { imageNotAvailable, metadataKvsToKv, MetadataPanel } from 'component/metadata';
import { thousandSeparate } from 'component/numberformatter';
import { RatingViewer } from 'component/rating';
import {
	createSensitivityAuthorizer,
	Sensitivity,
	SensitivityHeadsUp,
	sensitivityLabel,
} from 'component/sensitivity';
import { TabController } from 'component/tabcontroller';
import { CollectionTagView } from 'component/tags';
import { DefaultLabel, Glyphicon, Panel, tableClassStripedHover } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Result } from 'f61ui/component/result';
import { Timestamp } from 'f61ui/component/timestamp';
import { unrecognizedValue } from 'f61ui/utils';
import { browseUrl, collectionUrl, serverInfoUrl } from 'generated/frontend_uiroutes';
import {
	CollectionCreate,
	CollectionMove,
	CollectionRefreshMetadataAutomatically,
	CollectionTriggerMediaScan,
	DirectoryChangeDescription,
	DirectoryChangeReplicationPolicy,
	DirectoryChangeSensitivity,
	DirectoryCreate,
	DirectoryDelete,
	DirectoryMove,
	DirectoryPullTmdbMetadata,
	DirectoryRename,
	DirectorySetType,
} from 'generated/stoserver/stoservertypes_commands';
import { getDirectory } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CollectionSubset,
	CollectionSubsetWithMeta,
	DirectoryAndMeta,
	DirectoryOutput,
	DirectoryType,
	DocRef,
	MetadataImdbId,
} from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface BrowsePageProps {
	directoryId: string;
	view?: string; // defaults to autodetect
}

interface BrowsePageState {
	output: Result<DirectoryOutput>;
	selectedCollIds: string[];
}

// for decorate-sort-undecorate
interface DirOrCollection {
	name: string; // only used for sorting
	dir?: DirectoryAndMeta;
	coll?: CollectionSubsetWithMeta;
}

export default class BrowsePage extends React.Component<BrowsePageProps, BrowsePageState> {
	state: BrowsePageState = {
		output: new Result<DirectoryOutput>((_) => {
			this.setState({ output: _ });
		}),
		selectedCollIds: [],
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		const [output, loadingOrError] = this.state.output.unwrap();

		if (!output) {
			return (
				<AppDefaultLayout title="Loading" breadcrumbs={[]}>
					{loadingOrError}
				</AppDefaultLayout>
			);
		}

		const breadcrumbs: Breadcrumb[] = output.Parents.map((dirAndMeta) => {
			const dirx = dirAndMeta.Directory; // shorthand
			return {
				title: dirx.Name,
				url: browseUrl({ dir: dirx.Id, view: this.props.view }),
			};
		});

		const dir = output.Directory.Directory; // shorthand

		const showTabController =
			output.Collections.filter(hasMeta).length > 0 &&
			dir.Type !== DirectoryType.Movies &&
			dir.Type !== DirectoryType.Series;

		const content = ((): React.ReactNode => {
			switch (this.props.view) {
				case undefined: // = "auto"
					if (showTabController) {
						return this.richView(output);
					} else {
						return this.folderView(output);
					}
				case 'folder':
					return this.folderView(output);
				default:
					throw new Error(`unknown view: ${this.props.view}`);
			}
		})();

		return (
			<AppDefaultLayout
				title={dir.Name}
				titleElem={
					<span>
						{dir.Name}
						{dirDescription(output.Directory) && (
							<span className="margin-left">
								<DefaultLabel>{dirDescription(output.Directory)}</DefaultLabel>
							</span>
						)}
					</span>
				}
				breadcrumbs={breadcrumbs}>
				<SensitivityHeadsUp />
				<div className="row">
					<div className="col-md-9">
						<div>
							{output.Parents.map((dirAndMeta) => {
								return (
									dirAndMeta.MetaCollection && (
										<MetadataPanel collWithMeta={dirAndMeta.MetaCollection} />
									)
								);
							})}
							{output.Directory.MetaCollection && (
								<MetadataPanel collWithMeta={output.Directory.MetaCollection} />
							)}
						</div>

						{showTabController ? (
							<TabController
								tabs={[
									{
										url: browseUrl({
											dir: this.props.directoryId,
										}),
										title: 'Metadata view',
									},
									{
										url: browseUrl({
											dir: this.props.directoryId,
											view: 'folder',
										}),
										title: 'Folder view',
									},
								]}>
								{content}
							</TabController>
						) : (
							content
						)}
					</div>
					<div className="col-md-3">{this.directoryPanel(output)}</div>
				</div>
			</AppDefaultLayout>
		);
	}

	private folderView(output: DirectoryOutput): React.ReactNode {
		const dir = output.Directory.Directory; // shorthand

		const sensitivityAuthorize = createSensitivityAuthorizer();

		const masterCheckedChange = () => {
			const selectedCollIds = output.Collections.map((coll) => coll.Collection.Id);

			this.setState({ selectedCollIds });
		};

		const collCheckedChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			const collId = e.target.value;

			// removes collId if it already exists
			const selectedCollIds = this.state.selectedCollIds.filter((id) => id !== collId);

			if (e.target.checked) {
				selectedCollIds.push(collId);
			}

			this.setState({ selectedCollIds });
		};

		const collectionToRow = (collWithMeta: CollectionSubsetWithMeta) => {
			const coll = collWithMeta.Collection; // shorthand
			const dirIsForMovies = dir.Type === DirectoryType.Movies;
			const warning = dirIsForMovies &&
				!(MetadataImdbId in metadataKvsToKv(coll.Metadata)) && (
					<div>{metadataMissingIcon()}</div>
				);

			return (
				<tr key={coll.Id}>
					<td>
						<input
							type="checkbox"
							checked={this.state.selectedCollIds.indexOf(coll.Id) !== -1}
							onChange={collCheckedChange}
							value={coll.Id}
						/>
					</td>
					<td>
						<Glyphicon title="Collection" icon="duplicate" />
					</td>
					<td>
						{sensitivityAuthorize(coll.Sensitivity) ? (
							<div>
								<a
									href={collectionUrl({
										id: coll.Id,
									})}>
									{coll.Name}
								</a>
								{coll.Description && (
									<span className="margin-left">
										<DefaultLabel>{coll.Description}</DefaultLabel>
									</span>
								)}
								{coll.Sensitivity > Sensitivity.FamilyFriendly
									? mkSensitivityBadge(coll.Sensitivity)
									: null}
							</div>
						) : (
							<div>
								<span
									style={{
										color: 'transparent',
										textShadow: '0 0 7px rgba(0,0,0,0.5)',
									}}>
									{coll.Name}
								</span>
								{mkSensitivityBadge(coll.Sensitivity)}
							</div>
						)}
					</td>
					<td style={{ whiteSpace: 'nowrap' }}>
						{dirIsForMovies && <RatingViewer rating={coll.Rating} />}
						{warning} <CollectionTagView collection={coll} />
					</td>
					<td>{collectionDropdown(coll)}</td>
				</tr>
			);
		};

		const directoryToRow = (dirAndMeta: DirectoryAndMeta) => {
			const dirx = dirAndMeta.Directory;
			const content = sensitivityAuthorize(dirx.Sensitivity) ? (
				<div>
					<a href={browseUrl({ dir: dirx.Id, view: this.props.view })}>{dirx.Name}</a>
					{dirDescription(dirAndMeta) && (
						<span className="margin-left">
							<DefaultLabel>{dirDescription(dirAndMeta)}</DefaultLabel>
						</span>
					)}
					{dirx.Sensitivity > Sensitivity.FamilyFriendly &&
						mkSensitivityBadge(dirx.Sensitivity)}
				</div>
			) : (
				<div>
					<span style={{ color: 'transparent', textShadow: '0 0 7px rgba(0,0,0,0.5)' }}>
						{dirx.Name}
					</span>
					{mkSensitivityBadge(dirx.Sensitivity)}
				</div>
			);

			const dirIsForSeries = dirx.Type === DirectoryType.Series;
			const warning =
				dirIsForSeries &&
				(!dirAndMeta.MetaCollection ||
					!(
						MetadataImdbId in
						metadataKvsToKv(dirAndMeta.MetaCollection.Collection.Metadata)
					))
					? metadataMissingIcon()
					: null;

			return (
				<tr>
					<td />
					<td>
						<Glyphicon title="Directory" icon="folder-open" />
					</td>
					<td>{content}</td>
					<td>{warning}</td>
					<td>{directoryDropdown(dirAndMeta)}</td>
				</tr>
			);
		};

		const docToRow = (doc: DirOrCollection) => {
			if (doc.dir) {
				return directoryToRow(doc.dir);
			} else if (doc.coll) {
				return collectionToRow(doc.coll);
			}
			throw new Error('should not happen');
		};

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th style={{ width: '1%' }}>
							<input type="checkbox" onChange={masterCheckedChange} />
						</th>
						<th style={{ width: '1%' }} />
						<th />
						<th style={{ width: '1%' }} />
						<th style={{ width: '1%' }} />
					</tr>
				</thead>
				<tbody>{mergeDirectoriesAndCollectionsSorted(output).map(docToRow)}</tbody>
				<tfoot>
					<tr>
						<td colSpan={99}>
							{this.state.selectedCollIds.length > 0 && (
								<div>
									<CommandButton
										command={CollectionMove(this.state.selectedCollIds)}
									/>
									<CommandButton
										command={CollectionRefreshMetadataAutomatically(
											this.state.selectedCollIds,
										)}
									/>
								</div>
							)}

							{this.state.selectedCollIds.length === 0 && (
								<div>
									<span className="margin-right">
										<CommandButton command={DirectoryCreate(dir.Id)} />
									</span>
									<span className="margin-right">
										<CommandButton
											command={CollectionCreate(dir.Id, {
												redirect: (id) =>
													collectionUrl({
														id,
													}),
											})}
										/>
									</span>
								</div>
							)}
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	// FIXME: sensitivityAuthorize
	private richView(output: DirectoryOutput): React.ReactNode {
		return mergeDirectoriesAndCollectionsSorted(output).map((doc) => {
			if (doc.dir) {
				const dir = doc.dir.Directory;
				return (
					<Panel heading={dir.Name}>
						<a
							href={browseUrl({
								dir: dir.Id,
								view: this.props.view,
							})}>
							<img src={imageNotAvailable()} style={{ maxWidth: '100%' }} />
						</a>
					</Panel>
				);
			} else if (doc.coll) {
				return (
					<MetadataPanel
						collWithMeta={doc.coll}
						imageLinksToCollection={true}
						showTitle={true}
					/>
				);
			}
			throw new Error('should not happen');
		});
	}

	private directoryPanel(output: DirectoryOutput): React.ReactNode {
		const dir = output.Directory.Directory; // shorthand

		const dirHelp = helpByDirectoryType(dir.Type);

		return (
			<Panel
				heading={
					<div>
						Details &nbsp;
						{directoryDropdown(output.Directory)}
					</div>
				}>
				<table className={tableClassStripedHover}>
					<tbody>
						<tr>
							<th>ID</th>
							<td>
								{dir.Id}
								<span className="margin-left">
									<ClipboardButton text={dir.Id} />
								</span>
							</td>
						</tr>
						{dir.Type !== DirectoryType.Generic && (
							<tr>
								<th>Type</th>
								<td>{directoryTypeToEmoji(dir.Type)}</td>
							</tr>
						)}
						{dirHelp && (
							<tr>
								<th>Docs</th>
								<td>
									<DocLink doc={dirHelp.doc} title={dirHelp.title} />
								</td>
							</tr>
						)}
						<tr>
							<th>Content</th>
							<td>
								{thousandSeparate(output.SubDirectories.length)} subdirectories
								<br />
								{thousandSeparate(output.Collections.length)} collections
							</td>
						</tr>
						<tr>
							<th>Created</th>
							<td>
								<Timestamp ts={dir.Created} />
							</td>
						</tr>
						<tr>
							<th>Replication policy</th>
							<td>{resolveDirReplicationPolicy(output)}</td>
						</tr>
					</tbody>
				</table>
			</Panel>
		);
	}

	private fetchData() {
		this.state.output.load(() => getDirectory(this.props.directoryId));
	}
}

function mergeDirectoriesAndCollectionsSorted(output: DirectoryOutput): DirOrCollection[] {
	let docs: DirOrCollection[] = [];

	docs = docs.concat(output.SubDirectories.map((dir) => ({ name: dir.Directory.Name, dir })));
	docs = docs.concat(output.Collections.map((coll) => ({ name: coll.Collection.Name, coll })));

	docs.sort((a, b) => (a.name < b.name ? -1 : 1));

	return docs;
}

const mkSensitivityBadge = (sens: Sensitivity) => (
	// link to the page where we can upgrade sensitivity
	<a href={serverInfoUrl()}>
		<span className="badge margin-left">
			<Glyphicon icon="lock" />
			&nbsp;Level: {sensitivityLabel(sens)}
		</span>
	</a>
);

const directoryDropdown = (dirAndMeta: DirectoryAndMeta) => {
	const dir = dirAndMeta.Directory;
	const metaColl: CollectionSubset | null = dirAndMeta.MetaCollection
		? dirAndMeta.MetaCollection.Collection
		: null;

	return (
		<Dropdown>
			<CommandLink command={DirectoryRename(dir.Id, dir.Name)} />
			<CommandLink command={DirectoryChangeDescription(dir.Id, dirDescription(dirAndMeta))} />
			<CommandLink command={DirectoryChangeSensitivity(dir.Id, dir.Sensitivity)} />
			<CommandLink
				command={DirectoryChangeReplicationPolicy(dir.Id, dir.ReplicationPolicy || '', {
					Policy: replicationPolicyAutocomplete,
				})}
			/>
			<CommandLink
				command={DirectoryPullTmdbMetadata(dir.Id, { ForeignKey: tmdbTvAutocomplete })}
			/>
			{metaColl && <CommandLink command={CollectionTriggerMediaScan(metaColl.Id)} />}
			<CommandLink command={DirectoryMove(dir.Id, { disambiguation: dir.Name })} />
			<CommandLink
				command={DirectorySetType(dir.Id, dir.Type, { disambiguation: dir.Name })}
			/>
			<CommandLink command={DirectoryDelete(dir.Id, { disambiguation: dir.Name })} />
		</Dropdown>
	);
};

const hasMeta = (coll: CollectionSubsetWithMeta): boolean =>
	coll.Collection.Metadata && coll.Collection.Metadata.length > 0;

interface HelpForDirType {
	doc: DocRef;
	title: string;
}

function helpByDirectoryType(type: DirectoryType): HelpForDirType | null {
	switch (type) {
		case DirectoryType.Movies:
			return { doc: DocRef.DocsContentMoviesIndexMd, title: 'Guide: storing movies' };
		case DirectoryType.Series:
			return { doc: DocRef.DocsContentTvshowsIndexMd, title: 'Guide: storing TV shows' };
		case DirectoryType.Games:
			return { doc: DocRef.DocsContentGamesIndexMd, title: 'Guide: storing games' };
		case DirectoryType.Generic: // have doc, but let's not pollute every directory
		case DirectoryType.Podcasts:
		case DirectoryType.Albums:
			return null;
		default:
			throw unrecognizedValue(type);
	}
}

function directoryTypeToEmoji(type: DirectoryType): string {
	switch (type) {
		case DirectoryType.Generic:
			return '🗀';
		case DirectoryType.Movies:
			return '🎬';
		case DirectoryType.Series:
			return '📺';
		case DirectoryType.Podcasts:
			return '🎙️';
		case DirectoryType.Games:
			return '🎮';
		case DirectoryType.Albums:
			return '🏞️';
		default:
			throw unrecognizedValue(type);
	}
}

function metadataMissingIcon() {
	return <Glyphicon title="Metadata missing" icon="exclamation-sign" />;
}

// if not explicitly defined, walk parents until found
function resolveDirReplicationPolicy(output: DirectoryOutput): string {
	if (output.Directory.Directory.ReplicationPolicy) {
		return output.Directory.Directory.ReplicationPolicy;
	}

	const parentsReversed = output.Parents.slice().reverse();

	for (const parent of parentsReversed) {
		if (parent.Directory.ReplicationPolicy) {
			return parent.Directory.ReplicationPolicy;
		}
	}

	// should not happen - at the very least the root should have policy set
	throw new Error('unable to resolve ReplicationPolicy');
}

function dirDescription(dirAndMeta: DirectoryAndMeta): string {
	return dirAndMeta.MetaCollection ? dirAndMeta.MetaCollection.Collection.Description : '';
}
