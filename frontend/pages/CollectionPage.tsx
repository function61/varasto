import { tmdbMovieAutocomplete } from 'component/autocompletes';
import { AssetImg } from 'component/assetimg';
import { collectionDropdown } from 'component/collectiondropdown';
import { Filetype, filetypeForFile, iconForFiletype } from 'component/filetypes';
import { FileUploadArea } from 'component/fileupload';
import { metadataKvsToKv, MetadataPanel } from 'component/metadata';
import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'f61ui/component/result';
import { SensitivityHeadsUp } from 'component/sensitivity';
import { CollectionTagEditor } from 'component/tags';
import { RatingEditor } from 'component/rating';
import { InfoAlert } from 'f61ui/component/alerts';
import {
	DefaultLabel,
	tableClassStripedHover,
	AnchorButton,
	Glyphicon,
	Panel,
} from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandInlineForm } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Timestamp } from 'f61ui/component/timestamp';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	CollectionMoveFilesIntoAnotherCollection,
	CollectionDeleteFiles,
	CollectionPullTmdbMetadata,
} from 'generated/stoserver/stoservertypes_commands';
import {
	downloadFileUrl,
	getCollectiotAtRev,
	getConfig,
	getDirectory,
} from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgNetworkShareBaseUrl,
	ChangesetSubset,
	CollectionOutput,
	ConfigValue,
	Directory,
	DirectoryOutput,
	HeadRevisionId,
	DirectoryType,
	File as File2, // conflicts with HTML's "File" interface
	MetadataImdbId,
	RootPathDotBase64FIXME,
	CollectionSubset,
} from 'generated/stoserver/stoservertypes_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseUrl, collectionUrl } from 'generated/stoserver/stoserverui_uiroutes';

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

		if (!collectionOutput || !directoryOutput) {
			return (
				<AppDefaultLayout title="Loading" breadcrumbs={[]}>
					{loadingOrError}
				</AppDefaultLayout>
			);
		}

		const ret = this.renderBreadcrumbs(collectionOutput, directoryOutput);

		return (
			<AppDefaultLayout
				title={ret.title}
				titleElem={
					<span>
						{ret.title}
						{collectionOutput.Collection.Description && (
							<span className="margin-left">
								<DefaultLabel>
									{collectionOutput.Collection.Description}
								</DefaultLabel>
							</span>
						)}
					</span>
				}
				breadcrumbs={ret.breadcrumbs}>
				{loadingOrError}
				{this.renderData(collectionOutput, directoryOutput)}
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
			const dl = downloadFileUrl(
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
						<Glyphicon icon="folder-open" />
					</td>
					<td>
						<a
							href={collectionUrl({
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
					<td>{changeset.Id === collOutput.ChangesetId ? '→' : ''}</td>
					<td>
						<a
							href={collectionUrl({
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
			const dl = downloadFileUrl(
				collOutput.Collection.Id,
				collOutput.ChangesetId,
				file.Path,
			);

			const thumbUrl = `/api/thumbnails/thumb?coll=${collOutput.Collection.Id}&file=${file.Sha256}`;

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
				(dir) => dir.Type === DirectoryType.Movies || dir.Type === DirectoryType.Series,
			).length > 0;
		const imdbIdExpectedButMissing =
			inMoviesOrSeriesHierarchy && !(MetadataImdbId in metadataKv);

		return (
			<div>
				<SensitivityHeadsUp />

				{inMoviesOrSeriesHierarchy && this.nextPreviousButtons(collOutput, directoryOutput)}

				<div className="row">
					<div className="col-md-8">
						<MetadataPanel data={metadataKv} />

						{eligibleForThumbnail.length > 0 ? (
							<Panel heading="Thumbs">{eligibleForThumbnail.map(toThumbnail)}</Panel>
						) : null}

						<Panel heading="Files">
							<table className={tableClassStripedHover}>
								<thead>
									<tr>
										<td style={{ width: '1%' }} />
										<td style={{ width: '1%' }} />
										<td colSpan={99} />
									</tr>
								</thead>
								<tbody>
									{collOutput.SelectedPathContents.SubDirs.map(subDirToRow)}
									{collOutput.SelectedPathContents.Files.map(fileToRow)}
								</tbody>
							</table>

							{noFilesOrSubdirs && (
								<InfoAlert>Collection is currently empty.</InfoAlert>
							)}
						</Panel>

						{this.state.selectedFileHashes.length > 0 && (
							<div>
								<span className="margin-right">
									<CommandButton
										command={CollectionMoveFilesIntoAnotherCollection(
											collOutput.Collection.Id,
											this.state.selectedFileHashes.join(','),
										)}
									/>
								</span>
								<span className="margin-right">
									<CommandButton
										command={CollectionDeleteFiles(
											collOutput.Collection.Id,
											this.state.selectedFileHashes.join(','),
										)}
									/>
								</span>
							</div>
						)}
					</div>
					<div className="col-md-4">
						<Panel
							heading={
								<div>
									Details &nbsp;
									{collectionDropdown(collOutput.Collection)}
								</div>
							}>
							<table className={tableClassStripedHover}>
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
										<th>Rating</th>
										<td>
											<RatingEditor
												rating={collOutput.Collection.Rating}
												collId={collOutput.Collection.Id}
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
										<th>Replication policy</th>
										<td>{collOutput.Collection.ReplicationPolicy}</td>
									</tr>
									<tr>
										<th>
											Encryption keys{' '}
											<Info text="Usually has exactly one key. Additional keys appear if files are moved or deduplicated here from other collections (each have own encryption key)." />
										</th>
										<td
											title={collOutput.Collection.EncryptionKeyIds.join(
												', ',
											)}>
											Using {collOutput.Collection.EncryptionKeyIds.length}{' '}
											key(s)
										</td>
									</tr>
									<tr>
										<th>Clone command</th>
										<td>
											<ClipboardButton
												text={`sto clone ${collOutput.Collection.Id}`}
											/>
										</td>
									</tr>
									<tr>
										<th>FUSE &amp; network share</th>
										<td>
											{this.state.networkShareBaseUrl.draw(
												(networkShareBaseUrl) => {
													const networkSharePath =
														networkShareBaseUrl.Value +
														collOutput.Collection.Id +
														' - ' +
														collOutput.Collection.Name;

													return (
														<div title={networkSharePath}>
															<ClipboardButton
																text={networkSharePath}
															/>
														</div>
													);
												},
											)}
										</td>
									</tr>
								</tbody>
							</table>

							{imdbIdExpectedButMissing && (
								<CommandInlineForm
									command={CollectionPullTmdbMetadata(collOutput.Collection.Id, {
										ForeignKey: tmdbMovieAutocomplete,
									})}
								/>
							)}
						</Panel>

						<Panel
							heading={
								<div>
									Upload &nbsp;
									<Info text="You can upload one or more files by drag-n-dropping here. You can also use the 'Choose files' button." />
								</div>
							}>
							<FileUploadArea
								collectionId={this.props.id}
								collectionRevision={
									this.state.collectionOutputFixme!.Collection.Head
								}
							/>
						</Panel>

						<Panel heading="Changesets">
							<table className={tableClassStripedHover}>
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

	private renderBreadcrumbs(
		collectionOutput: CollectionOutput,
		directoryOutput: DirectoryOutput,
	) {
		const dirToBreadcrumb = (dir: Directory): Breadcrumb => {
			return {
				title: dir.Name,
				url: browseUrl({ dir: dir.Id, view: '' }),
			};
		};

		const parentDirToBreadcrumb = (pd: string): Breadcrumb => {
			return {
				title: pd,
				url: collectionUrl({
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
				url: collectionUrl({
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

	private nextPreviousButtons(collOutput: CollectionOutput, directoryOutput: DirectoryOutput) {
		// given collection ID, extracts previous/next sibling from directoryOutput
		const nextPrevious = (
			id: string,
		): { prev: CollectionSubset | null; next: CollectionSubset | null } => {
			for (let i = 0; i < directoryOutput.Collections.length; i++) {
				if (directoryOutput.Collections[i].Id !== id) {
					continue;
				}
				return {
					prev: i > 0 ? directoryOutput.Collections[i - 1] : null,
					next:
						i + 1 < directoryOutput.Collections.length
							? directoryOutput.Collections[i + 1]
							: null,
				};
			}

			return { prev: null, next: null };
		};

		const np = nextPrevious(collOutput.Collection.Id);

		// if neither, it's no sense to show stub UI
		if (!np.prev && !np.next) {
			return null;
		}

		return (
			<div
				className="btn-group"
				role="group"
				aria-label="Next / previous"
				style={{ marginBottom: '16px' }}>
				{np.prev && (
					<AnchorButton href={collectionUrlDefault(np.prev.Id)}>
						<Glyphicon icon="chevron-left" />
						&nbsp;
						{np.prev.Name}
					</AnchorButton>
				)}
				<span className="btn btn-default disabled">{collOutput.Collection.Name}</span>
				{np.next && (
					<AnchorButton href={collectionUrlDefault(np.next.Id)}>
						{np.next.Name}
						&nbsp;
						<Glyphicon icon="chevron-right" />
					</AnchorButton>
				)}
			</div>
		);
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

// 'subdir/subsubdir/foo.txt' => 'foo.txt'
// 'foo.txt' => 'foo.txt'
function filenameFromPath(path: string): string {
	return /\/?([^/]+)$/.exec(path)![1];
}

function collectionUrlDefault(id: string) {
	return collectionUrl({
		id,
		rev: HeadRevisionId,
		path: RootPathDotBase64FIXME,
	});
}
