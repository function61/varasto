import { AssetImg } from 'component/assetimg';
import { ClipboardButton } from 'component/clipboardbutton';
import { Filetype, filetypeForFile, iconForFiletype } from 'component/filetypes';
import { thousandSeparate } from 'component/numberformatter';
import { SensitivityHeadsUp } from 'component/sensitivity';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { CommandButton } from 'f61ui/component/CommandButton';
import { Info } from 'f61ui/component/info';
import { Loading } from 'f61ui/component/loading';
import { Timestamp } from 'f61ui/component/timestamp';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { CollectionMoveFilesIntoAnotherCollection } from 'generated/varastoserver_commands';
import {
	downloadFileUrl,
	getCollectiotAtRev,
	getDirectory,
} from 'generated/varastoserver_endpoints';
import {
	ChangesetSubset,
	CollectionOutput,
	Directory,
	DirectoryOutput,
	File,
	RootPathDotBase64FIXME,
} from 'generated/varastoserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseRoute, collectionRoute } from 'routes';

interface CollectionPageProps {
	id: string;
	rev: string;
	pathBase64: string;
}

interface CollectionPageState {
	collectionOutput?: CollectionOutput;
	directoryOutput?: DirectoryOutput;
	selectedFilePaths: string[];
}

export default class CollectionPage extends React.Component<
	CollectionPageProps,
	CollectionPageState
> {
	state: CollectionPageState = { selectedFilePaths: [] };

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		const collectionOutput = this.state.collectionOutput;
		const directoryOutput = this.state.directoryOutput;

		let breadcrumbs: Breadcrumb[] = [];
		let title = 'Loading';

		if (collectionOutput && directoryOutput) {
			const ret = this.renderBreadcrumbs(collectionOutput, directoryOutput);
			title = ret.title;
			breadcrumbs = ret.breadcrumbs;
		}

		return (
			<AppDefaultLayout title={title} breadcrumbs={breadcrumbs}>
				{collectionOutput ? this.renderData(collectionOutput) : <Loading />}
			</AppDefaultLayout>
		);
	}

	private renderData(collOutput: CollectionOutput) {
		const eligibleForThumbnail = collOutput.SelectedPathContents.Files.filter(
			(file) => filetypeForFile(file) === Filetype.Picture,
		);

		const fileCheckedChange = (e: React.ChangeEvent<HTMLInputElement>) => {
			// remove from currently selected, so depending on checked we can add or not add it
			const selectedFilePaths = this.state.selectedFilePaths.filter(
				(sel) => sel !== e.target.value,
			);

			if (e.target.checked) {
				selectedFilePaths.push(e.target.value);
			}

			this.setState({ selectedFilePaths });
		};

		const fileToRow = (file: File) => {
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
							checked={this.state.selectedFilePaths.indexOf(file.Path) !== -1}
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
						<a href={dl} target="_new">
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

		const changesetToItem = (changeset: ChangesetSubset) => (
			<tr>
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

		const changesetsReversed = collOutput.Collection.Changesets.slice().reverse();

		const toThumbnail = (file: File) => {
			const dl = downloadUrlFIXME(
				collOutput.Collection.Id,
				collOutput.ChangesetId,
				file.Path,
			);

			const thumb = `http://localhost:8688/thumb?coll=${collOutput.Collection.Id}&amp;file=${
				file.Sha256
			}`;

			return (
				<a href={dl} target="_blank" title={file.Path} className="margin-left">
					<img src={thumb} className="img-thumbnail" />
				</a>
			);
		};

		return (
			<div>
				<SensitivityHeadsUp />
				<div className="row">
					<div className="col-md-8">
						{eligibleForThumbnail.length > 0 ? (
							<Panel heading="Thumbs">{eligibleForThumbnail.map(toThumbnail)}</Panel>
						) : (
							''
						)}

						<Panel heading="Files">
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
						</Panel>

						{this.state.selectedFilePaths.length > 0 ? (
							<CommandButton
								command={CollectionMoveFilesIntoAnotherCollection(
									collOutput.Collection.Id,
									this.state.selectedFilePaths.join(','),
								)}
							/>
						) : (
							''
						)}
					</div>
					<div className="col-md-4">
						<Panel heading="Details">
							<table className="table table-striped table-hover">
								<tbody>
									<tr>
										<th>Changeset</th>
										<td>{collOutput.ChangesetId}</td>
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
												text={`varasto clone ${collOutput.Collection.Id}`}
											/>
										</td>
									</tr>
								</tbody>
							</table>
						</Panel>
						<Panel heading="Changesets">
							<table className="table table-striped table-hover">
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
				url: browseRoute.buildUrl({ dir: dir.Id }),
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
		const collectionOutput = await getCollectiotAtRev(
			this.props.id,
			this.props.rev,
			this.props.pathBase64,
		);

		const directoryOutput = await getDirectory(collectionOutput.Collection.Directory);

		this.setState({ collectionOutput, directoryOutput });
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
