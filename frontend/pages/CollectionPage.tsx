import { AssetImg } from 'component/assetimg';
import { bytesToHumanReadable } from 'component/bytesformatter';
import { Info } from 'component/info';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { Loading } from 'f61ui/component/loading';
import { Timestamp } from 'f61ui/component/timestamp';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getCollectiotAtRev, getDirectory } from 'generated/bupserver_endpoints';
import {
	ChangesetSubset,
	CollectionOutput,
	Directory,
	DirectoryOutput,
	File,
} from 'generated/bupserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseRoute, collectionRoute, replicationPoliciesRoute } from 'routes';

interface CollectionPageProps {
	id: string;
	rev: string;
	pathBase64: string;
}

interface CollectionPageState {
	collectionOutput?: CollectionOutput;
	directoryOutput?: DirectoryOutput;
}

const rootPathFIXME = 'Lg==';

export default class CollectionPage extends React.Component<
	CollectionPageProps,
	CollectionPageState
> {
	state: CollectionPageState = {};

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
		const fileToRow = (file: File) => {
			const dl = downloadUrl(collOutput.Collection.Id, collOutput.ChangesetId, file.Path);

			return (
				<tr>
					<td>
						<AssetImg src="/file.png" />
					</td>
					<td>
						<a href={dl} target="_new">
							{filenameFromPath(file.Path)}
						</a>
					</td>
					<td>
						<Timestamp ts={file.Created} />
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
					<td>
						<AssetImg src="/directory.png" />
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

		return (
			<div className="row">
				<div className="col-md-8">
					<Panel heading="Files">
						<table className="table table-striped table-hover">
							<tbody>
								{collOutput.SelectedPathContents.SubDirs.map(subDirToRow)}
								{collOutput.SelectedPathContents.Files.map(fileToRow)}
							</tbody>
						</table>
					</Panel>
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
									<td>{collOutput.FileCount}</td>
								</tr>
								<tr>
									<th>
										Total size <Info text="at selected revision" />
									</th>
									<td>{bytesToHumanReadable(collOutput.TotalSize)}</td>
								</tr>
								<tr>
									<th>Replication policy</th>
									<td>
										<a href={replicationPoliciesRoute.buildUrl({})}>
											{collOutput.Collection.ReplicationPolicy}
										</a>
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
					path: rootPathFIXME,
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

function downloadUrl(collectionId: string, changesetId: string, path: string): string {
	return `/collections/${collectionId}/rev/${changesetId}/dl?file=` + encodeURIComponent(path);
}

// 'subdir/subsubdir/foo.txt' => 'foo.txt'
// 'foo.txt' => 'foo.txt'
function filenameFromPath(path: string): string {
	return /\/?([^/]+)$/.exec(path)![1];
}
