import { AssetImg } from 'component/assetimg';
import { collectionDropdown } from 'component/collectiondropdown';
import { Filetype, filetypeForFile, iconForFiletype } from 'component/filetypes';
import { metadataKvsToKv, MetadataPanel } from 'component/metadata';
import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'component/result';
import { SensitivityHeadsUp } from 'component/sensitivity';
import { CollectionTagEditor } from 'component/tags';
import { reloadCurrentPage } from 'f61ui/browserutils';
import { InfoAlert } from 'f61ui/component/alerts';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandInlineForm } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Timestamp } from 'f61ui/component/timestamp';
import { httpMustBeOk, makeQueryParams } from 'f61ui/httputil';
import { dateObjToDateTime } from 'f61ui/types';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	CollectionFuseMount,
	CollectionMoveFilesIntoAnotherCollection,
	CollectionPullMetadata,
} from 'generated/stoserver/stoservertypes_commands';
import {
	commitChangeset,
	downloadFileUrl,
	generateIds,
	getCollectiotAtRev,
	getConfig,
	getDirectory,
	uploadFileUrl,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgNetworkShareBaseUrl,
	ChangesetSubset,
	CollectionOutput,
	ConfigValue,
	Directory,
	DirectoryOutput,
	File as File2,
	MetadataImdbId, // conflicts with HTML's "File" interface
	RootPathDotBase64FIXME,
} from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseRoute, collectionRoute } from 'routes';

// FIXME
const moviesDirId = '70MqRF3FaxI';
const seriesDirId = '7JczPh5-XSQ';

interface CollectionPageProps {
	id: string;
	rev: string;
	pathBase64: string;
}

interface CollectionPageState {
	collectionOutput: Result<CollectionOutput>;
	collectionOutputFixme?: CollectionOutput; // for file uploads, think again
	directoryOutput: Result<DirectoryOutput>;
	networkShareBaseUrl: Result<ConfigValue>;
	selectedFileHashes: string[];
}

export default class CollectionPage extends React.Component<
	CollectionPageProps,
	CollectionPageState
> {
	state: CollectionPageState = {
		collectionOutput: new Result<CollectionOutput>((_) => {
			this.setState({ collectionOutput: _ });
		}),
		directoryOutput: new Result<DirectoryOutput>((_) => {
			this.setState({ directoryOutput: _ });
		}),
		networkShareBaseUrl: new Result<ConfigValue>((_) => {
			this.setState({ networkShareBaseUrl: _ });
		}),
		selectedFileHashes: [],
	};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		const [collectionOutput, directoryOutput, loadingOrError] = Result.unwrap2(
			this.state.collectionOutput,
			this.state.directoryOutput,
		);

		let breadcrumbs: Breadcrumb[] = [];
		let title = 'Loading';

		if (collectionOutput && directoryOutput) {
			const ret = this.renderBreadcrumbs(collectionOutput, directoryOutput);
			title = ret.title;
			breadcrumbs = ret.breadcrumbs;
		}

		return (
			<AppDefaultLayout title={title} breadcrumbs={breadcrumbs}>
				{loadingOrError}
				{collectionOutput && directoryOutput
					? this.renderData(collectionOutput, directoryOutput)
					: null}
			</AppDefaultLayout>
		);
	}

	private renderData(collOutput: CollectionOutput, directoryOutput: DirectoryOutput) {
		const eligibleForThumbnail = collOutput.SelectedPathContents.Files.filter(
			(file) => filetypeForFile(file) === Filetype.Picture,
		);

		const fileCheckedChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			// remove from currently selected, so depending on checked we can add or not add it
			const selectedFileHashes = this.state.selectedFileHashes.filter(
				(sel) => sel !== e.target.value,
			);

			if (e.target.checked) {
				selectedFileHashes.push(e.target.value);
			}

			this.setState({ selectedFileHashes });
		};

		const fileToRow = (file: File2) => {
			const dl = downloadUrlFIXME(
				collOutput.Collection.Id,
				collOutput.ChangesetId,
				file.Path,
			);

			return (
				<tr key={file.Path}>
					<td>
						<input
							type="checkbox"
							onChange={fileCheckedChange}
							checked={this.state.selectedFileHashes.indexOf(file.Sha256) !== -1}
							value={file.Sha256}
						/>
					</td>
					<td>
						<AssetImg
							width={22}
							height={22}
							src={'/filetypes/' + iconForFiletype(filetypeForFile(file))}
						/>
					</td>
					<td>
						<a href={dl} target="_blank">
							{filenameFromPath(file.Path)}
						</a>
					</td>
					<td>
						<Timestamp ts={file.Modified} />
					</td>
					<td>{bytesToHumanReadable(file.Size)}</td>
				</tr>
			);
		};

		const subDirToRow = (subDir: string) => {
			return (
				<tr>
					<td />
					<td>
						<span className="glyphicon glyphicon-folder-open" />
					</td>
					<td>
						<a
							href={collectionRoute.buildUrl({
								id: this.props.id,
								rev: this.props.rev,
								path: btoa(subDir),
							})}>
							{filenameFromPath(subDir)}/
						</a>
					</td>
					<td colSpan={99} />
				</tr>
			);
		};

		const changesetToItem = (changeset: ChangesetSubset) => {
			return (
				<tr>
					<td>{changeset.Id === collOutput.ChangesetId ? '*' : ''}</td>
					<td>
						<a
							href={collectionRoute.buildUrl({
								id: collOutput.Collection.Id,
								rev: changeset.Id,
								path: this.props.pathBase64,
							})}>
							{changeset.Id}
						</a>
					</td>
					<td>
						<Timestamp ts={changeset.Created} />
					</td>
				</tr>
			);
		};

		const changesetsReversed = collOutput.Collection.Changesets.slice().reverse();

		const toThumbnail = (file: File2) => {
			const dl = downloadUrlFIXME(
				collOutput.Collection.Id,
				collOutput.ChangesetId,
				file.Path,
			);

			const thumbUrl =
				'//' +
				document.location!.hostname +
				`:8688/thumb?coll=${collOutput.Collection.Id}&amp;file=${file.Sha256}`;

			return (
				<a href={dl} target="_blank" title={file.Path} className="margin-left">
					<img src={thumbUrl} className="img-thumbnail" />
				</a>
			);
		};

		const noFilesOrSubdirs =
			collOutput.SelectedPathContents.SubDirs.length +
				collOutput.SelectedPathContents.Files.length ===
			0;

		const metadataKv = metadataKvsToKv(collOutput.Collection.Metadata);

		const inMoviesOrSeriesHierarchy =
			directoryOutput.Parents.concat(directoryOutput.Directory).filter(
				(dir) => [moviesDirId, seriesDirId].indexOf(dir.Id) !== -1,
			).length > 0;
		const imdbIdExpectedButMissing =
			inMoviesOrSeriesHierarchy && !(MetadataImdbId in metadataKv);

		return (
			<div>
				<SensitivityHeadsUp />
				<div className="row">
					<div className="col-md-8">
						<MetadataPanel data={metadataKv} />

						{eligibleForThumbnail.length > 0 ? (
							<Panel heading="Thumbs">{eligibleForThumbnail.map(toThumbnail)}</Panel>
						) : null}

						<Panel
							heading={
								<div>
									Files
									{collOutput.Collection.Description ? (
										<span className="label label-default margin-left">
											{collOutput.Collection.Description}
										</span>
									) : null}
								</div>
							}>
							<table className="table table-striped table-hover">
								<thead>
									<tr>
										<td style={{ width: '1%' }} />
										<td colSpan={99} />
									</tr>
								</thead>
								<tbody>
									{collOutput.SelectedPathContents.SubDirs.map(subDirToRow)}
									{collOutput.SelectedPathContents.Files.map(fileToRow)}
								</tbody>
							</table>

							{noFilesOrSubdirs ? (
								<InfoAlert>Collection is currently empty.</InfoAlert>
							) : null}
						</Panel>

						<Panel heading="Upload">
							<input
								type="file"
								id="upload"
								multiple={true}
								onChange={(e) => {
									this.fileChange(e);
								}}
							/>
						</Panel>

						{this.state.selectedFileHashes.length > 0 ? (
							<CommandButton
								command={CollectionMoveFilesIntoAnotherCollection(
									collOutput.Collection.Id,
									this.state.selectedFileHashes.join(','),
								)}
							/>
						) : null}
					</div>
					<div className="col-md-4">
						<Panel
							heading={
								<div>
									Details &nbsp;
									{collectionDropdown(collOutput.Collection)}
								</div>
							}>
							<table className="table table-striped table-hover">
								<tbody>
									<tr>
										<th>Id</th>
										<td>
											{collOutput.Collection.Id}

											<ClipboardButton text={collOutput.Collection.Id} />
										</td>
									</tr>
									<tr>
										<th>Tags</th>
										<td>
											<CollectionTagEditor
												collection={collOutput.Collection}
											/>
										</td>
									</tr>
									<tr>
										<th>Created</th>
										<td>
											<Timestamp ts={collOutput.Collection.Created} />
										</td>
									</tr>
									<tr>
										<th>File count</th>
										<td>{thousandSeparate(collOutput.FileCount)}</td>
									</tr>
									<tr>
										<th>
											Total size <Info text="at selected revision" />
										</th>
										<td>{bytesToHumanReadable(collOutput.TotalSize)}</td>
									</tr>
									<tr>
										<th>Desired volumes</th>
										<td>{collOutput.Collection.DesiredVolumes.join(', ')}</td>
									</tr>
									<tr>
										<th>Clone command</th>
										<td>
											<ClipboardButton
												text={`sto clone ${collOutput.Collection.Id}`}
											/>
										</td>
									</tr>
								</tbody>
							</table>

							{imdbIdExpectedButMissing ? (
								<CommandInlineForm
									command={CollectionPullMetadata(collOutput.Collection.Id)}
								/>
							) : null}
						</Panel>
						{this.state.networkShareBaseUrl.draw((networkShareBaseUrl) => (
							<Panel
								heading={
									<div>
										FUSE &amp; network share{' '}
										<ClipboardButton text={networkShareBaseUrl.Value} />
									</div>
								}>
								<CommandButton
									command={CollectionFuseMount(collOutput.Collection.Id)}
								/>
							</Panel>
						))}
						<Panel heading="Changesets">
							<table className="table table-striped table-hover">
								<thead>
									<tr>
										<td style={{ width: '1%' }} />
										<td />
										<td />
									</tr>
								</thead>
								<tbody>{changesetsReversed.map(changesetToItem)}</tbody>
							</table>
						</Panel>
					</div>
				</div>
			</div>
		);
	}

	private fileChange(e: React.ChangeEvent<HTMLInputElement>) {
		if (!e.target.files || e.target.files.length === 0) {
			return;
		}

		// is not an array, so convert to one
		const files: File[] = [];
		// tslint:disable-next-line:prefer-for-of
		for (let i = 0; i < e.target.files.length; ++i) {
			files.push(e.target.files[i]);
		}

		shouldAlwaysSucceed(this.uploadAllFiles(files));
	}

	private async uploadAllFiles(files: File[]) {
		const createdFiles: File2[] = [];
		for (const file of files) {
			createdFiles.push(await this.uploadOneFile(file));
		}

		await commitChangeset(this.props.id, {
			ID: (await generateIds()).Changeset,
			Parent: this.state.collectionOutputFixme!.Collection.Head,
			Created: dateObjToDateTime(new Date()),
			FilesCreated: createdFiles,
			FilesUpdated: [],
			FilesDeleted: [],
		});

		reloadCurrentPage();
	}

	private async uploadOneFile(file: File): Promise<File2> {
		const uploadEndpoint = makeQueryParams(
			uploadFileUrl(this.state.collectionOutputFixme!.Collection.Id),
			{
				mtime: file.lastModified.toString(),
				filename: file.name, // spec says this is "without path information"
			},
		);

		const result: File2 = await fetch(uploadEndpoint, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/octet-stream',
			},
			body: file,
		})
			.then(httpMustBeOk)
			.then((response) => response.json());

		return result;
	}

	private renderBreadcrumbs(
		collectionOutput: CollectionOutput,
		directoryOutput: DirectoryOutput,
	) {
		const dirToBreadcrumb = (dir: Directory): Breadcrumb => {
			return {
				title: dir.Name,
				url: browseRoute.buildUrl({ dir: dir.Id, v: '' }),
			};
		};

		const parentDirToBreadcrumb = (pd: string): Breadcrumb => {
			return {
				title: pd,
				url: collectionRoute.buildUrl({
					id: this.props.id,
					rev: this.props.rev,
					path: btoa(pd),
				}),
			};
		};

		const areWeNavigatedToSubdir = collectionOutput.SelectedPathContents.Path !== '.';

		const collName = collectionOutput.Collection.Name + ' 📚';

		const title = areWeNavigatedToSubdir
			? filenameFromPath(collectionOutput.SelectedPathContents.Path)
			: collName;

		// path leading to our repo
		let breadcrumbs = directoryOutput.Parents.map(dirToBreadcrumb);

		breadcrumbs.push(dirToBreadcrumb(directoryOutput.Directory));

		// collection name
		if (areWeNavigatedToSubdir) {
			breadcrumbs.push({
				title: collName,
				url: collectionRoute.buildUrl({
					id: this.props.id,
					rev: this.props.rev,
					path: RootPathDotBase64FIXME,
				}),
			});
		}

		breadcrumbs = breadcrumbs.concat(
			collectionOutput.SelectedPathContents.ParentDirs.map(parentDirToBreadcrumb),
		);

		return {
			breadcrumbs,
			title,
		};
	}

	private async fetchData() {
		this.state.networkShareBaseUrl.load(() => getConfig(CfgNetworkShareBaseUrl));

		const collectionOutputPromise = getCollectiotAtRev(
			this.props.id,
			this.props.rev,
			this.props.pathBase64,
		);

		this.state.collectionOutput.load(() => collectionOutputPromise);

		const collectionOutput = await collectionOutputPromise;

		this.setState({ collectionOutputFixme: collectionOutput });

		this.state.directoryOutput.load(() => getDirectory(collectionOutput.Collection.Directory));
	}
}

function downloadUrlFIXME(collectionId: string, changesetId: string, path: string): string {
	// FIXME: this is a hack
	return downloadFileUrl(collectionId, changesetId) + '?file=' + encodeURIComponent(path);
}

// 'subdir/subsubdir/foo.txt' => 'foo.txt'
// 'foo.txt' => 'foo.txt'
function filenameFromPath(path: string): string {
	return /\/?([^/]+)$/.exec(path)![1];
}
