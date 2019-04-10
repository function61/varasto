import { ClipboardButton } from 'component/clipboardbutton';
import { getMaxSensitivityFromLocalStorage, SensitivityHeadsUp } from 'component/sensitivity';
import { Panel } from 'f61ui/component/bootstrap';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	CollectionChangeDescription,
	CollectionDelete,
	CollectionFuseMount,
	CollectionMove,
	CollectionRename,
	DirectoryChangeDescription,
	DirectoryChangeSensitivity,
	DirectoryCreate,
	DirectoryDelete,
	DirectoryMove,
	DirectoryRename,
} from 'generated/varastoserver_commands';
import { getDirectory } from 'generated/varastoserver_endpoints';
import {
	CollectionSubset,
	Directory,
	DirectoryOutput,
	HeadRevisionId,
	RootPathDotBase64FIXME,
} from 'generated/varastoserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseRoute, collectionRoute } from 'routes';

interface BrowsePageProps {
	directoryId: string;
}

interface BrowsePageState {
	output?: DirectoryOutput;
}

// for decorate-sort-undecorate
interface DirOrCollection {
	name: string; // only used for sorting
	dir?: Directory;
	coll?: CollectionSubset;
}

export default class BrowsePage extends React.Component<BrowsePageProps, BrowsePageState> {
	state: BrowsePageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		const showMaxSensitivity = getMaxSensitivityFromLocalStorage();

		const collectionToRow = (coll: CollectionSubset) => (
			<tr>
				<td>
					<span title="Collection" className="glyphicon glyphicon-duplicate" />
				</td>
				<td>
					<a
						href={collectionRoute.buildUrl({
							id: coll.Id,
							rev: HeadRevisionId,
							path: RootPathDotBase64FIXME,
						})}>
						{coll.Name}
					</a>
					{coll.Description ? (
						<span className="label label-default margin-left">{coll.Description}</span>
					) : (
						''
					)}
				</td>
				<td>
					<Dropdown>
						<CommandLink command={CollectionRename(coll.Id, coll.Name)} />
						<CommandLink
							command={CollectionChangeDescription(coll.Id, coll.Description)}
						/>
						<CommandLink command={CollectionMove(coll.Id)} />
						<CommandLink command={CollectionFuseMount(coll.Id)} />
						<CommandLink command={CollectionDelete(coll.Id)} />
					</Dropdown>
				</td>
			</tr>
		);

		const directoryToRow = (dir: Directory) => {
			const sensitivityBadge = (
				<span className="badge margin-left">
					<span className="glyphicon glyphicon-lock" />
					&nbsp;Level: {dir.Sensitivity}
				</span>
			);

			const content =
				dir.Sensitivity <= showMaxSensitivity ? (
					<div>
						<a href={browseRoute.buildUrl({ dir: dir.Id })}>{dir.Name}</a>
						{dir.Description ? (
							<span className="label label-default margin-left">
								{dir.Description}
							</span>
						) : (
							''
						)}
						{dir.Sensitivity > 0 ? sensitivityBadge : ''}
					</div>
				) : (
					<div>
						<span
							style={{ color: 'transparent', textShadow: '0 0 7px rgba(0,0,0,0.5)' }}>
							{dir.Name}
						</span>
						{sensitivityBadge}
					</div>
				);

			return (
				<tr>
					<td>
						<span title="Directory" className="glyphicon glyphicon-folder-open" />
					</td>
					<td>{content}</td>
					<td>
						<Dropdown>
							<CommandLink command={DirectoryRename(dir.Id, dir.Name)} />
							<CommandLink
								command={DirectoryChangeDescription(dir.Id, dir.Description)}
							/>
							<CommandLink
								command={DirectoryChangeSensitivity(dir.Id, dir.Sensitivity)}
							/>
							<CommandLink command={DirectoryMove(dir.Id)} />
							<CommandLink command={DirectoryDelete(dir.Id)} />
						</Dropdown>
					</td>
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

		const output = this.state.output;

		let title = 'Loading';
		let breadcrumbs: Breadcrumb[] = [];

		if (output) {
			title = output.Directory.Name;
			breadcrumbs = output.Parents.map((dir) => {
				return {
					title: dir.Name,
					url: browseRoute.buildUrl({ dir: dir.Id }),
				};
			});
		}

		return (
			<AppDefaultLayout title={title} breadcrumbs={breadcrumbs}>
				{!output ? (
					<Loading />
				) : (
					<div>
						<SensitivityHeadsUp />
						<div className="row">
							<div className="col-md-9">
								<table className="table table-striped table-hover">
									<thead>
										<tr>
											<th style={{ width: '1%' }} />
											<th />
											<th style={{ width: '1%' }} />
										</tr>
									</thead>
									<tbody>
										{mergeDirectoriesAndCollectionsSorted(output).map(docToRow)}
									</tbody>
									<tfoot>
										<tr>
											<td colSpan={99}>
												<CommandButton
													command={DirectoryCreate(output.Directory.Id)}
												/>
											</td>
										</tr>
									</tfoot>
								</table>
							</div>
							<div className="col-md-3">
								<Panel heading={`Directory: ${output.Directory.Name}`}>
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
							</div>
						</div>
					</div>
				)}
			</AppDefaultLayout>
		);
	}

	private async fetchData() {
		const output = await getDirectory(this.props.directoryId);

		this.setState({ output });
	}
}

function mergeDirectoriesAndCollectionsSorted(output: DirectoryOutput): DirOrCollection[] {
	let docs: DirOrCollection[] = [];

	docs = docs.concat(output.Directories.map((dir) => ({ name: dir.Name, dir })));
	docs = docs.concat(output.Collections.map((coll) => ({ name: coll.Name, coll })));

	docs.sort((a, b) => (a.name < b.name ? -1 : 1));

	return docs;
}
