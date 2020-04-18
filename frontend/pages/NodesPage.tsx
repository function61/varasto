import { Result } from 'f61ui/component/result';
import { WarningAlert } from 'f61ui/component/alerts';
import { CommandLink } from 'f61ui/component/CommandButton';
import { tableClassStripedHover, Panel } from 'f61ui/component/bootstrap';
import { Dropdown } from 'f61ui/component/dropdown';
import { Info } from 'f61ui/component/info';
import { Timestamp } from 'f61ui/component/timestamp';
import {
	NodeInstallTlsCert,
	NodeChangeSmartBackend,
} from 'generated/stoserver/stoservertypes_commands';
import { getNodes } from 'generated/stoserver/stoservertypes_endpoints';
import { Node } from 'generated/stoserver/stoservertypes_types';
import { SettingsLayout } from 'layout/settingslayout';
import * as React from 'react';

interface NodesPageState {
	nodes: Result<Node[]>;
}

export default class NodesPage extends React.Component<{}, NodesPageState> {
	state: NodesPageState = {
		nodes: new Result<Node[]>((_) => {
			this.setState({ nodes: _ });
		}),
	};

	componentDidMount() {
		this.fetchData();
	}

	componentWillReceiveProps() {
		this.fetchData();
	}

	render() {
		return (
			<SettingsLayout title="Servers" breadcrumbs={[]}>
				<Panel heading="Servers">{this.renderData()}</Panel>
				<Panel heading="Info">{this.info()}</Panel>
			</SettingsLayout>
		);
	}

	private renderData() {
		const [nodes, loadingOrError] = this.state.nodes.unwrap();

		return (
			<table className={tableClassStripedHover}>
				<thead>
					<tr>
						<th>Id</th>
						<th>Addr</th>
						<th>Name</th>
						<th>TLS cert</th>
						<th>TLS cert expires</th>
						<th />
					</tr>
				</thead>
				<tbody>
					{(nodes || []).map((node: Node) => (
						<tr key={node.Id}>
							<td>{node.Id}</td>
							<td>{node.Addr}</td>
							<td>{node.Name}</td>
							<td>
								{node.TlsCert.Identity}{' '}
								<Info
									text={`Issuer: ${node.TlsCert.Issuer}\nAlgo: ${node.TlsCert.PublicKeyAlgorithm}`}
								/>
							</td>
							<td>
								<Timestamp ts={node.TlsCert.NotAfter} />
							</td>
							<td>
								<Dropdown>
									<CommandLink command={NodeInstallTlsCert(node.Id)} />
									<CommandLink command={NodeChangeSmartBackend(node.Id)} />
								</Dropdown>
							</td>
						</tr>
					))}
				</tbody>
				<tfoot>{loadingOrError}</tfoot>
			</table>
		);
	}

	private fetchData() {
		this.state.nodes.load(() => getNodes());
	}

	private info() {
		return (
			<div>
				<p>
					Having multiple servers gives you high availability - if one of your servers
					crashes/goes offline, you can still access your data.
				</p>
				<WarningAlert>💰 High availability requires a paid license.</WarningAlert>
			</div>
		);
	}
}
