import { bytesToHumanReadable } from 'f61ui/component/bytesformatter';
import { Loading } from 'f61ui/component/loading';
import { shouldAlwaysSucceed } from 'f61ui/utils';
import { getServerInfo } from 'generated/varastoserver_endpoints';
import { ServerInfo } from 'generated/varastoserver_types';
import { AppDefaultLayout } from 'layout/appdefaultlayout';
import * as React from 'react';

interface ServerInfoPageState {
	serverInfo?: ServerInfo;
}

export default class ServerInfoPage extends React.Component<{}, ServerInfoPageState> {
	state: ServerInfoPageState = {};

	componentDidMount() {
		shouldAlwaysSucceed(this.fetchData());
	}

	componentWillReceiveProps() {
		shouldAlwaysSucceed(this.fetchData());
	}

	render() {
		return (
			<AppDefaultLayout title="Server info" breadcrumbs={[]}>
				{this.renderData()}
			</AppDefaultLayout>
		);
	}

	private renderData() {
		const serverInfo = this.state.serverInfo;

		if (!serverInfo) {
			return <Loading />;
		}

		interface Item {
			h: string;
			t: string;
		}

		const items: Item[] = [
			{ h: 'Varasto version', t: serverInfo.AppVersion },
			{ h: 'Go version', t: serverInfo.GoVersion },
			{ h: 'Server OS / arch', t: `${serverInfo.ServerOs} / ${serverInfo.ServerArch}` },
			{ h: 'Goroutines', t: serverInfo.Goroutines.toString() },
			{ h: 'Heap memory', t: bytesToHumanReadable(serverInfo.HeapBytes) },
		];

		return (
			<table className="table table-striped table-hover">
				<tbody>
					{items.map((item) => (
						<tr>
							<th>{item.h}</th>
							<td>{item.t}</td>
						</tr>
					))}
				</tbody>
			</table>
		);
	}

	private async fetchData() {
		const serverInfo = await getServerInfo();

		this.setState({ serverInfo });
	}
}
