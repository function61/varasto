import { collectionDropdown } from 'component/collectiondropdown';
import { metadataKvsToKv, MetadataPanel } from 'component/metadata';
import { Result } from 'component/result';
import {
	createSensitivityAuthorizer,
	Sensitivity,
	SensitivityHeadsUp,
	sensitivityLabel,
} from 'component/sensitivity';
import { TabController } from 'component/tabcontroller';
import { CollectionTagView } from 'component/tags';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { globalConfig } from 'f61ui/globalconfig';
import {
	CollectionCreate,
	CollectionMove,
	CollectionRefreshMetadataAutomatically,
	DirectoryChangeDescription,
	DirectoryChangeSensitivity,
	DirectoryCreate,
	DirectoryDelete,
	DirectoryMove,
	DirectoryPullMetadata,
	DirectoryRename,
} from 'generated/stoserver/stoservertypes_commands';
import { getDirectory } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CollectionSubset,
	Directory,
	DirectoryOutput,
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
import { browseRoute, collectionRoute, serverInfoRoute } from 'routes';

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

// FIXME
const moviesDirId = '70MqRF3FaxI';
const seriesDirId = '7JczPh5-XSQ';

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
				url: browseRoute.buildUrl({ dir: dir.Id, v: this.props.view }),
			};
		});

		const showTabController =
			output.Collections.filter(hasMeta).length > 0 && output.Directory.Id !== moviesDirId;

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
			<AppDefaultLayout title={output.Directory.Name} breadcrumbs={breadcrumbs}>
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
										url: browseRoute.buildUrl({
											dir: this.props.directoryId,
											v: '',
										}),
										title: 'Metadata view',
									},
									{
										url: browseRoute.buildUrl({
											dir: this.props.directoryId,
											v: 'folder',
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
			const dirIsForMovies = coll.Directory === moviesDirId;
			const warning =
				dirIsForMovies && !(MetadataImdbId in metadataKvsToKv(coll.Metadata)) ? (
					<div>
						<span
							title="Metadata missing"
							className="glyphicon glyphicon-exclamation-sign"
						/>
					</div>
				) : null;

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
						<span title="Collection" className="glyphicon glyphicon-duplicate" />
					</td>
					<td>
						{sensitivityAuthorize(coll.Sensitivity) ? (
							<div>
								<a
									href={collectionRoute.buildUrl({
										id: coll.Id,
										rev: HeadRevisionId,
										path: RootPathDotBase64FIXME,
									})}>
									{coll.Name}
								</a>
								{coll.Description ? (
									<span className="label label-default margin-left">
										{coll.Description}
									</span>
								) : null}
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
					<td>
						{warning} <CollectionTagView collection={coll} />
					</td>
					<td>{collectionDropdown(coll)}</td>
				</tr>
			);
		};

		const directoryToRow = (dir: Directory) => {
			const content = sensitivityAuthorize(dir.Sensitivity) ? (
				<div>
					<a href={browseRoute.buildUrl({ dir: dir.Id, v: this.props.view })}>
						{dir.Name}
					</a>
					{dir.Description ? (
						<span className="label label-default margin-left">{dir.Description}</span>
					) : null}
					{dir.Sensitivity > Sensitivity.FamilyFriendly
						? mkSensitivityBadge(dir.Sensitivity)
						: null}
				</div>
			) : (
				<div>
					<span style={{ color: 'transparent', textShadow: '0 0 7px rgba(0,0,0,0.5)' }}>
						{dir.Name}
					</span>
					{mkSensitivityBadge(dir.Sensitivity)}
				</div>
			);

			const dirIsForSeries = dir.Parent === seriesDirId;
			const warning =
				dirIsForSeries && !(MetadataImdbId in metadataKvsToKv(dir.Metadata)) ? (
					<span
						title="Metadata missing"
						className="glyphicon glyphicon-exclamation-sign"
					/>
				) : null;

			return (
				<tr>
					<td />
					<td>
						<span title="Directory" className="glyphicon glyphicon-folder-open" />
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
			<table className="table table-striped table-hover">
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

							<CommandButton command={CollectionCreate(output.Directory.Id)} />
						</td>
					</tr>
				</tfoot>
			</table>
		);
	}

	private richView(output: DirectoryOutput): React.ReactNode {
		const collectionToRow = (coll: CollectionSubset): React.ReactNode => {
			const metadata = metadataKvsToKv(coll.Metadata);

			const imageSrc = metadata[MetadataThumbnail] || imageNotAvailable();

			const badges = [];

			if (MetadataReleaseDate in metadata) {
				badges.push(
					<span className="label label-default">📅 {metadata[MetadataReleaseDate]}</span>,
				);
			}

			return (
				<Panel
					heading={
						<div>
							{coll.Name} - {metadata[MetadataTitle] || ''} &nbsp;
							{badges}
						</div>
					}>
					<a
						href={collectionRoute.buildUrl({
							id: coll.Id,
							rev: HeadRevisionId,
							path: RootPathDotBase64FIXME,
						})}>
						<img
							title={metadata[MetadataOverview] || ''}
							src={imageSrc}
							style={{ maxWidth: '100%' }}
						/>
					</a>
				</Panel>
			);
		};

		const directoryToRow = (dir: Directory): React.ReactNode => {
			return (
				<Panel heading={dir.Name}>
					<a
						href={browseRoute.buildUrl({
							dir: dir.Id,
							v: this.props.view,
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
		return (
			<Panel
				heading={
					<div>
						Details &nbsp;
						{directoryDropdown(output.Directory)}
					</div>
				}>
				<table className="table table-striped table-hover">
					<tbody>
						<tr>
							<th>Id</th>
							<td>
								{output.Directory.Id}
								<ClipboardButton text={output.Directory.Id} />
							</td>
						</tr>
						<tr>
							<th>Content</th>
							<td>
								{output.Directories.length} subdirectories
								<br />
								{output.Collections.length} collections
							</td>
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
	<a href={serverInfoRoute.buildUrl({})}>
		<span className="badge margin-left">
			<span className="glyphicon glyphicon-lock" />
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
			<CommandLink command={DirectoryPullMetadata(dir.Id)} />
			<CommandLink command={DirectoryMove(dir.Id, { disambiguation: dir.Name })} />
			<CommandLink command={DirectoryDelete(dir.Id, { disambiguation: dir.Name })} />
		</Dropdown>
	);
};

const hasMeta = (coll: CollectionSubset): boolean => coll.Metadata && coll.Metadata.length > 0;

function imageNotAvailable(): string {
	return globalConfig().assetsDir + '/../image-not-available.png';
}
