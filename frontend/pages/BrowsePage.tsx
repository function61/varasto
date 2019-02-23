import { AssetImg } from 'component/assetimg';
import { Breadcrumb } from 'f61ui/component/breadcrumbtrail';
import { CommandButton, CommandLink } from 'f61ui/component/CommandButton';
import { Dropdown } from 'f61ui/component/dropdown';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import {
	CollectionMove,
	CollectionRename,
	DirectoryCreate,
	DirectoryRename,
} from 'generated/bupserver_commands';
import { getDirectory } from 'generated/bupserver_endpoints';
import {
	CollectionSubset,
	Directory,
	DirectoryOutput,
	HeadRevisionId,
} from 'generated/bupserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';
import { browseRoute, collectionRoute } from 'routes';

interface BrowsePageProps {
	directoryId: string;
}

interface BrowsePageState {
	output?: DirectoryOutput;
}

const rootPathFIXME = 'Lg==';

export default class BrowsePage extends React.Component<BrowsePageProps, BrowsePageState> {
	state: BrowsePageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		const collectionToRow = (coll: CollectionSubset) => (
			<tr>
				<td>
					<AssetImg src="/collection.png" />
				</td>
				<td>
					<a
						href={collectionRoute.buildUrl({
							id: coll.Id,
							rev: HeadRevisionId,
							path: rootPathFIXME,
						})}>
						{coll.Name}
					</a>
				</td>
				<td>
					<Dropdown>
						<CommandLink command={CollectionRename(coll.Id, coll.Name)} />
						<CommandLink command={CollectionMove(coll.Id)} />
					</Dropdown>
				</td>
			</tr>
		);

		const directoryToRow = (dir: Directory) => (
			<tr>
				<td>
					<AssetImg src="/directory.png" />
				</td>
				<td>
					<a href={browseRoute.buildUrl({ dir: dir.Id })}>{dir.Name}</a>
				</td>
				<td>
					<Dropdown>
						<CommandLink command={DirectoryRename(dir.Id, dir.Name)} />
					</Dropdown>
				</td>
			</tr>
		);

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
					<table className="table table-striped table-hover">
						<tbody>
							{output.Directories.map(directoryToRow)}
							{output.Collections.map(collectionToRow)}
						</tbody>
						<tfoot>
							<tr>
								<td colSpan={99}>
									<CommandButton command={DirectoryCreate(output.Directory.Id)} />
								</td>
							</tr>
						</tfoot>
					</table>
				)}
			</AppDefaultLayout>
		);
	}

	private async fetchData() {
		const output = await getDirectory(this.props.directoryId);

		this.setState({ output });
	}
}
