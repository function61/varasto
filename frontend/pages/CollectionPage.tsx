import { tmdbMovieAutocomplete } from 'component/autocompletes';
import { AssetImg } from 'component/assetimg';
import { collectionDropdown } from 'component/collectiondropdown';
import { relativeDateFormat, shouldAlwaysSucceed } from 'f61ui/utils';
import { filetypeForFile, iconForFiletype } from 'component/filetypes';
import { FileUploadArea } from 'component/fileupload';
import { metadataKvsToKv, MetadataPanel, imageNotAvailable } from 'component/metadata';
import { thousandSeparate } from 'component/numberformatter';
import { Result } from 'f61ui/component/result';
import { SensitivityHeadsUp } from 'component/sensitivity';
import { CollectionTagEditor } from 'component/tags';
import { RatingEditor } from 'component/rating';
import { Pager, PagerData } from 'f61ui/component/pager';
import { InfoAlert } from 'f61ui/component/alerts';
import {
	DefaultLabel,
	tableClassStripedHover,
	AnchorButton,
	Glyphicon,
	Panel,
	GridRowMaker,
} from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { ClipboardButton } from 'f61ui/component/clipboardbutton';
import { CommandButton, CommandInlineForm } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Timestamp } from 'f61ui/component/timestamp';
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
	DirectoryAndMeta,
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
import { browseUrl, collectionUrl } from 'generated/frontend_uiroutes';

interface CollectionPageProps {
	id: string;
	rev?: string; // default: head revision
	page?: number; // default: 1st
	view?: string; // default: autodetect
	pathBase64?: string; // default: root
}

interface CollectionPageState {
	collectionOutput: Result<CollectionOutput>;
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
		// without this, after using CollectionMoveFilesIntoAnotherCollection and
		// CollectionDeleteFiles, the old selections would persist after page reload
		this.setState({ selectedFileHashes: [] });

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
						{collectionOutput.CollectionWithMeta.Collection.Description && (
							<span className="margin-left">
								<DefaultLabel>
									{collectionOutput.CollectionWithMeta.Collection.Description}
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
		const coll = collOutput.CollectionWithMeta.Collection; // shorthand

		const eligibleForThumbnail = collOutput.SelectedPathContents.Files.filter(
			(file) =>
				collOutput.CollectionWithMeta.FilesInMeta.indexOf(makeThumbPath(file.Sha256)) !==
				-1,
		);

		const changesetToItem = (changeset: ChangesetSubset) => {
			return (
				<tr>
					<td>{changeset.Id === collOutput.ChangesetId ? '→' : ''}</td>
					<td>
						<a
							href={collectionUrl({
								id: coll.Id,
								rev: changeset.Id,
								path: this.props.pathBase64,
								view: this.props.view,
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

		const changesetsReversed = coll.Changesets.slice().reverse();

		const metadataKv = metadataKvsToKv(coll.Metadata);

		const inMoviesOrSeriesHierarchy =
			directoryOutput.Parents.concat(directoryOutput.Directory).filter(
				(dir) =>
					dir.Directory.Type === DirectoryType.Movies ||
					dir.Directory.Type === DirectoryType.Series,
			).length > 0;
		const inSeriesHierarchy =
			directoryOutput.Parents.concat(directoryOutput.Directory).filter(
				(dir) => dir.Directory.Type === DirectoryType.Series,
			).length > 0;
		const imdbIdExpectedButMissing =
			inMoviesOrSeriesHierarchy && !(MetadataImdbId in metadataKv);

		const haveAnyThumbnails = eligibleForThumbnail.length > 0;

		return (
			<div>
				<SensitivityHeadsUp />

				<div className="row">
					<div className="col-md-8">
						<MetadataPanel
							collWithMeta={collOutput.CollectionWithMeta}
							showDetails={true}
						/>

						{(this.props.view === undefined && haveAnyThumbnails) ||
						this.props.view === 'thumb'
							? this.thumbnailView(collOutput)
							: this.listView(collOutput)}
					</div>
					<div className="col-md-4">
						{inSeriesHierarchy && this.nextPreviousButtons(collOutput, directoryOutput)}

						<Panel
							heading={
								<div>
									Details &nbsp;
									{collectionDropdown(coll)}
								</div>
							}>
							<table className={tableClassStripedHover}>
								<tbody>
									<tr>
										<th>Id</th>
										<td>
											{coll.Id}

											<span className="margin-left">
												<ClipboardButton text={coll.Id} />
											</span>
										</td>
									</tr>
									<tr>
										<th>Tags</th>
										<td>
											<CollectionTagEditor collection={coll} />
										</td>
									</tr>
									<tr>
										<th>Rating</th>
										<td>
											<RatingEditor rating={coll.Rating} collId={coll.Id} />
										</td>
									</tr>
									<tr>
										<th>Created</th>
										<td>
											<Timestamp ts={coll.Created} />
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
										<td>{coll.ReplicationPolicy}</td>
									</tr>
									<tr>
										<th>
											Encryption keys{' '}
											<Info text="Usually has exactly one key. Additional keys appear if files are moved or deduplicated here from other collections (each have own encryption key)." />
										</th>
										<td title={coll.EncryptionKeyIds.join(', ')}>
											Using {coll.EncryptionKeyIds.length} key(s)
										</td>
									</tr>
									<tr>
										<th>Clone command</th>
										<td>
											<ClipboardButton text={`sto clone ${coll.Id}`} />
										</td>
									</tr>
									<tr>
										<th>FUSE &amp; network share</th>
										<td>
											{this.state.networkShareBaseUrl.draw(
												(networkShareBaseUrl) => {
													const networkSharePath =
														networkShareBaseUrl.Value +
														coll.Id +
														' - ' +
														coll.Name;

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
									command={CollectionPullTmdbMetadata(coll.Id, {
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
								collectionRevision={coll.Head}
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

	private listView(collOutput: CollectionOutput): JSX.Element {
		const coll = collOutput.CollectionWithMeta.Collection; // shorthand

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
			const dl = downloadFileUrl(coll.Id, collOutput.ChangesetId, file.Path);

			return (
				<tr key={file.Path}>
					<td>
						<input
							type="checkbox"
							onChange={fileCheckedChange}
							checked={this.state.selectedFileHashes.indexOf(file.Path) !== -1}
							value={file.Path}
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
								view: this.props.view,
							})}>
							{filenameFromPath(subDir)}/
						</a>
					</td>
					<td colSpan={99} />
				</tr>
			);
		};

		const noFilesOrSubdirs =
			collOutput.SelectedPathContents.SubDirs.length +
				collOutput.SelectedPathContents.Files.length ===
			0;

		return (
			<div>
				<div className="clearfix margin-bottom">
					<span className="pull-right">{this.thumbVsListViewSwitcher(false)}</span>
				</div>

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

					{noFilesOrSubdirs && <InfoAlert>Collection is currently empty.</InfoAlert>}
				</Panel>

				{this.state.selectedFileHashes.length > 0 && (
					<div>
						<span className="margin-right">
							<CommandButton
								command={CollectionMoveFilesIntoAnotherCollection(
									coll.Id,
									this.state.selectedFileHashes,
								)}
							/>
						</span>
						<span className="margin-right">
							<CommandButton
								command={CollectionDeleteFiles(
									coll.Id,
									this.state.selectedFileHashes,
								)}
							/>
						</span>
					</div>
				)}
			</div>
		);
	}

	private thumbnailView(collOutput: CollectionOutput): JSX.Element {
		const coll = collOutput.CollectionWithMeta.Collection; // shorthand

		const fileToThumbnail = (file: File2) => {
			const dl = downloadFileUrl(coll.Id, collOutput.ChangesetId, file.Path);

			const thumbPath = makeThumbPath(file.Sha256);

			const thumbUrl =
				collOutput.CollectionWithMeta.FilesInMeta.indexOf(thumbPath) !== -1
					? downloadFileUrl(
							coll.Id,
							collOutput.CollectionWithMeta.FilesInMetaAt,
							thumbPath,
					  )
					: imageNotAvailable();

			return (
				<div className="thumbnail">
					<a
						href={dl}
						target="_blank"
						title={
							relativeDateFormat(file.Modified) +
							' ' +
							bytesToHumanReadable(file.Size)
						}>
						<img src={thumbUrl} />
					</a>
					<div className="caption">
						<p>{file.Path}</p>
					</div>
				</div>
			);
		};

		const subDirToThumbnail = (subDir: string) => {
			return (
				<div className="thumbnail">
					<a
						href={collectionUrl({
							id: this.props.id,
							rev: this.props.rev,
							path: btoa(subDir),
						})}>
						<img src={imageNotAvailable()} />
					</a>
					<div className="caption">
						<p>{filenameFromPath(subDir)}/</p>
					</div>
				</div>
			);
		};

		const pagerData = new PagerData(
			this.props.page !== undefined ? this.props.page : 1,
			collOutput.SelectedPathContents.Files.length,
			30,
		);

		const pagerPageUrlMaker = (idx: number) =>
			collectionUrl({
				id: coll.Id,
				rev: this.props.rev,
				path: this.props.pathBase64,
				page: idx,
			});

		const rows = new GridRowMaker(3);

		interface FileOrSubdir {
			file?: File2;
			subDir?: string;
		}

		collOutput.SelectedPathContents.SubDirs.map(
			(subDir): FileOrSubdir => {
				return { subDir };
			},
		)
			.concat(
				collOutput.SelectedPathContents.Files.map(
					(file): FileOrSubdir => {
						return { file };
					},
				),
			)
			.filter(pagerData.idxFilter())
			.forEach((fos) => {
				if (fos.file) {
					rows.push(fileToThumbnail(fos.file));
				} else if (fos.subDir) {
					rows.push(subDirToThumbnail(fos.subDir));
				} else {
					throw new Error('Should not happen');
				}
			});

		return (
			<div>
				<div className="clearfix margin-bottom">
					<span className="pull-left">
						<Pager data={pagerData} pageUrl={pagerPageUrlMaker} />
					</span>
					<span className="pull-right">{this.thumbVsListViewSwitcher(true)}</span>
				</div>
				{rows.finalize()}
				<Pager data={pagerData} pageUrl={pagerPageUrlMaker} />
			</div>
		);
	}

	private renderBreadcrumbs(
		collectionOutput: CollectionOutput,
		directoryOutput: DirectoryOutput,
	) {
		const dirToBreadcrumb = (dirAndMeta: DirectoryAndMeta): Breadcrumb => {
			const dir = dirAndMeta.Directory;

			return {
				title: dir.Name,
				url: browseUrl({ dir: dir.Id }),
			};
		};

		const parentDirToBreadcrumb = (pd: string): Breadcrumb => {
			return {
				title: pd,
				url: collectionUrl({
					id: this.props.id,
					rev: this.props.rev,
					path: btoa(pd),
					view: this.props.view,
				}),
			};
		};

		const areWeNavigatedToSubdir = collectionOutput.SelectedPathContents.Path !== '.';

		const collName = collectionOutput.CollectionWithMeta.Collection.Name + ' 📚';

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
					view: this.props.view,
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
				if (directoryOutput.Collections[i].Collection.Id !== id) {
					continue;
				}
				return {
					prev: i > 0 ? directoryOutput.Collections[i - 1].Collection : null,
					next:
						i + 1 < directoryOutput.Collections.length
							? directoryOutput.Collections[i + 1].Collection
							: null,
				};
			}

			return { prev: null, next: null };
		};

		const np = nextPrevious(collOutput.CollectionWithMeta.Collection.Id);

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
					<AnchorButton href={collectionUrl({ id: np.prev.Id })}>
						<Glyphicon icon="chevron-left" />
						&nbsp;
						{np.prev.Name}
					</AnchorButton>
				)}
				<span className="btn btn-default disabled">
					{collOutput.CollectionWithMeta.Collection.Name}
				</span>
				{np.next && (
					<AnchorButton href={collectionUrl({ id: np.next.Id })}>
						{np.next.Name}
						&nbsp;
						<Glyphicon icon="chevron-right" />
					</AnchorButton>
				)}
			</div>
		);
	}

	private thumbVsListViewSwitcher(currentlyInThumbView: boolean): JSX.Element {
		const pickClass = (active: boolean) =>
			active ? 'btn btn-primary disabled' : 'btn btn-default';

		return (
			<div className="btn-group" role="group" aria-label="View type">
				<a
					className={pickClass(currentlyInThumbView)}
					href={collectionUrl({
						id: this.props.id,
						rev: this.props.rev,
						path: this.props.pathBase64,
						view: 'thumb',
					})}>
					Thumbnail view
				</a>
				<a
					className={pickClass(!currentlyInThumbView)}
					href={collectionUrl({
						id: this.props.id,
						rev: this.props.rev,
						path: this.props.pathBase64,
						view: 'list',
					})}>
					List view
				</a>
			</div>
		);
	}

	private async fetchData() {
		this.state.networkShareBaseUrl.load(() => getConfig(CfgNetworkShareBaseUrl));

		const collectionOutputPromise = getCollectiotAtRev(
			this.props.id,
			this.props.rev || HeadRevisionId,
			this.props.pathBase64 || RootPathDotBase64FIXME,
		);

		this.state.collectionOutput.load(() => collectionOutputPromise);

		const collectionOutput = await collectionOutputPromise;

		this.state.directoryOutput.load(() =>
			getDirectory(collectionOutput.CollectionWithMeta.Collection.Directory),
		);
	}
}

// 'subdir/subsubdir/foo.txt' => 'foo.txt'
// 'foo.txt' => 'foo.txt'
function filenameFromPath(path: string): string {
	return /\/?([^/]+)$/.exec(path)![1];
}

function makeThumbPath(sha256: string): string {
	return '.sto/thumb/' + sha256.substr(0, 10) + '.jpg';
}
