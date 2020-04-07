import { collectionDropdown } from 'component/collectiondropdown';
import { DocLink } from 'component/doclink';
import { metadataKvsToKv, MetadataPanel } from 'component/metadata';
import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'f61ui/component/result';
import {
	createSensitivityAuthorizer,
	Sensitivity,
	SensitivityHeadsUp,
	sensitivityLabel,
} from 'component/sensitivity';
import { TabController } from 'component/tabcontroller';
import { CollectionTagView } from 'component/tags';
import { RatingViewer } from 'component/rating';
import { tableClassStripedHover, DefaultLabel, Glyphicon, Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { globalConfig } from 'f61ui/globalconfig';
import { unrecognizedValue } from 'f61ui/utils';
import {
	CollectionCreate,
	CollectionMove,
	CollectionRefreshMetadataAutomatically,
	DirectoryChangeDescription,
	DirectoryChangeSensitivity,
	DirectoryCreate,
	DirectoryDelete,
	DirectoryChangeReplicationPolicy,
	DirectoryMove,
	DirectoryPullMetadata,
	DirectoryRename,
	DirectorySetType,
} from 'generated/stoserver/stoservertypes_commands';
import { getDirectory } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CollectionSubset,
	Directory,
	DirectoryOutput,
	DirectoryType,
	DocRef,
	HeadRevisionId,
	MetadataImdbId,
	MetadataOverview,
	MetadataReleaseDate,
	MetadataThumbnail,
	MetadataTitle,
	RootPathDotBase64FIXME,
} from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseUrl, collectionUrl, serverInfoUrl } from 'generated/stoserver/stoserverui_uiroutes';

interface BrowsePageProps {
	directoryId: string;
	view: string;
}

interface BrowsePageState {
	output: Result<DirectoryOutput>;
	selectedCollIds: string[];
}

// for decorate-sort-undecorate
interface DirOrCollection {
	name: string; // only used for sorting
	dir?: Directory;
	coll?: CollectionSubset;
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

		const breadcrumbs: Breadcrumb[] = output.Parents.map((dir) => {
			return {
				title: dir.Name,
				url: browseUrl({ dir: dir.Id, view: this.props.view }),
			};
		});

		const showTabController =
			output.Collections.filter(hasMeta).length > 0 &&
			output.Directory.Type !== DirectoryType.Movies &&
			output.Directory.Type !== DirectoryType.Series;

		const content = ((): React.ReactNode => {
			switch (this.props.view) {
				case '': // = "auto"
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
				title={output.Directory.Name}
				titleElem={
					<span>
						{output.Directory.Name}
						{output.Directory.Description && (
							<span className="margin-left">
								<DefaultLabel>{output.Directory.Description}</DefaultLabel>
							</span>
						)}
					</span>
				}
				breadcrumbs={breadcrumbs}>
				<SensitivityHeadsUp />
				<div className="row">
					<div className="col-md-9">
						<div>
							{output.Parents.map((dir: Directory) => (
								<MetadataPanel data={metadataKvsToKv(dir.Metadata)} />
							))}
							<MetadataPanel data={metadataKvsToKv(output.Directory.Metadata)} />
						</div>

						{showTabController ? (
							<TabController
								tabs={[
									{
										url: browseUrl({
											dir: this.props.directoryId,
											view: '',
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
		const selectedCollIdsSerialized = this.state.selectedCollIds.join(',');

		const sensitivityAuthorize = createSensitivityAuthorizer();

		const masterCheckedChange = () => {
			const selectedCollIds = output.Collections.map((coll) => coll.Id);

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

		const collectionToRow = (coll: CollectionSubset) => {
			const dirIsForMovies = output.Directory.Type === DirectoryType.Movies;
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
										rev: HeadRevisionId,
										path: RootPathDotBase64FIXME,
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

		const directoryToRow = (dir: Directory) => {
			const content = sensitivityAuthorize(dir.Sensitivity) ? (
				<div>
					<a href={browseUrl({ dir: dir.Id, view: this.props.view })}>{dir.Name}</a>
					{dir.Description && (
						<span className="margin-left">
							<DefaultLabel>{dir.Description}</DefaultLabel>
						</span>
					)}
					{dir.Sensitivity > Sensitivity.FamilyFriendly &&
						mkSensitivityBadge(dir.Sensitivity)}
				</div>
			) : (
				<div>
					<span style={{ color: 'transparent', textShadow: '0 0 7px rgba(0,0,0,0.5)' }}>
						{dir.Name}
					</span>
					{mkSensitivityBadge(dir.Sensitivity)}
				</div>
			);

			const dirIsForSeries = output.Directory.Type === DirectoryType.Series;
			const warning =
				dirIsForSeries && !(MetadataImdbId in metadataKvsToKv(dir.Metadata))
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
					<td>{directoryDropdown(dir)}</td>
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
							{selectedCollIdsSerialized ? (
								<div>
									<CommandButton
										command={CollectionMove(selectedCollIdsSerialized)}
									/>
									<CommandButton
										command={CollectionRefreshMetadataAutomatically(
											selectedCollIdsSerialized,
										)}
									/>
								</div>
							) : null}

							<CommandButton command={DirectoryCreate(output.Directory.Id)} />

							<CommandButton
								command={CollectionCreate(output.Directory.Id, {
									redirect: (id) =>
										collectionUrl({
											id,
											rev: HeadRevisionId,
											path: RootPathDotBase64FIXME,
										}),
								})}
							/>
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	// FIXME: sensitivityAuthorize
	private richView(output: DirectoryOutput): React.ReactNode {
		const collectionToRow = (coll: CollectionSubset): React.ReactNode => {
			const metadata = metadataKvsToKv(coll.Metadata);

			const imageSrc = metadata[MetadataThumbnail] || imageNotAvailable();

			const badges = [];

			if (MetadataReleaseDate in metadata) {
				badges.push(<DefaultLabel>📅 {metadata[MetadataReleaseDate]}</DefaultLabel>);
			}

			if (coll.Description) {
				badges.push(
					<span className="margin-left">
						<DefaultLabel>{coll.Description}</DefaultLabel>
					</span>,
				);
			}

			return (
				<Panel
					heading={
						<div>
							{coll.Name} - {metadata[MetadataTitle] || ''} &nbsp;
							{badges}
							<CollectionTagView collection={coll} />
						</div>
					}
					footer={metadata[MetadataOverview] && <p>{metadata[MetadataOverview]}</p>}>
					<a
						href={collectionUrl({
							id: coll.Id,
							rev: HeadRevisionId,
							path: RootPathDotBase64FIXME,
						})}>
						<img src={imageSrc} style={{ maxWidth: '100%' }} />
					</a>
				</Panel>
			);
		};

		const directoryToRow = (dir: Directory): React.ReactNode => {
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
		};

		const docToRow = (doc: DirOrCollection): React.ReactNode => {
			if (doc.dir) {
				return directoryToRow(doc.dir);
			} else if (doc.coll) {
				return collectionToRow(doc.coll);
			}
			throw new Error('should not happen');
		};

		return mergeDirectoriesAndCollectionsSorted(output).map(docToRow);
	}

	private directoryPanel(output: DirectoryOutput): React.ReactNode {
		const dirHelp = helpByDirectoryType(output.Directory.Type);

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
							<th>Id</th>
							<td>
								{output.Directory.Id}
								<ClipboardButton text={output.Directory.Id} />
							</td>
						</tr>
						{output.Directory.Type !== DirectoryType.Generic && (
							<tr>
								<th>Type</th>
								<td>{directoryTypeToEmoji(output.Directory.Type)}</td>
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
								{thousandSeparate(output.Directories.length)} subdirectories
								<br />
								{thousandSeparate(output.Collections.length)} collections
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

	docs = docs.concat(output.Directories.map((dir) => ({ name: dir.Name, dir })));
	docs = docs.concat(output.Collections.map((coll) => ({ name: coll.Name, coll })));

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

const directoryDropdown = (dir: Directory) => {
	return (
		<Dropdown>
			<CommandLink command={DirectoryRename(dir.Id, dir.Name)} />
			<CommandLink command={DirectoryChangeDescription(dir.Id, dir.Description)} />
			<CommandLink command={DirectoryChangeSensitivity(dir.Id, dir.Sensitivity)} />
			<CommandLink
				command={DirectoryChangeReplicationPolicy(dir.Id, dir.ReplicationPolicy || '')}
			/>
			<CommandLink command={DirectoryPullMetadata(dir.Id)} />
			<CommandLink command={DirectoryMove(dir.Id, { disambiguation: dir.Name })} />
			<CommandLink
				command={DirectorySetType(dir.Id, dir.Type, { disambiguation: dir.Name })}
			/>
			<CommandLink command={DirectoryDelete(dir.Id, { disambiguation: dir.Name })} />
		</Dropdown>
	);
};

const hasMeta = (coll: CollectionSubset): boolean => coll.Metadata && coll.Metadata.length > 0;

function imageNotAvailable(): string {
	return globalConfig().assetsDir + '/../image-not-available.png';
}

interface HelpForDirType {
	doc: DocRef;
	title: string;
}

function helpByDirectoryType(type: DirectoryType): HelpForDirType | null {
	switch (type) {
		case DirectoryType.Movies:
			return { doc: DocRef.DocsGuideStoringMoviesMd, title: 'Guide: storing movies' };
		case DirectoryType.Series:
			return { doc: DocRef.DocsGuideStoringTvshowsMd, title: 'Guide: storing TV shows' };
		case DirectoryType.Generic:
		case DirectoryType.Podcasts:
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
		default:
			throw unrecognizedValue(type);
	}
}

function metadataMissingIcon() {
	return <Glyphicon title="Metadata missing" icon="exclamation-sign" />;
}

// if not explicitly defined, walk parents until found
function resolveDirReplicationPolicy(output: DirectoryOutput): string {
	if (output.Directory.ReplicationPolicy) {
		return output.Directory.ReplicationPolicy;
	}

	const parentsReversed = output.Parents.reverse();

	for (const parent of parentsReversed) {
		if (parent.ReplicationPolicy) {
			return parent.ReplicationPolicy;
		}
	}

	// should not happen - at the very least the root should have policy set
	throw new Error('unable to resolve ReplicationPolicy');
}
