import { DocLink } from 'component/doclink';
import { Result } from 'f61ui/component/result';
import { Panel } from 'f61ui/component/bootstrap';
import { CommandButton, CommandIcon } from 'f61ui/component/CommandButton';
import {
	ConfigSetFuseServerBaseurl,
	ConfigSetNetworkShareBaseUrl,
	FuseUnmountAll,
} from 'generated/stoserver/stoservertypes_commands';
import { getConfig } from 'generated/stoserver/stoservertypes_endpoints';
import {
	CfgFuseServerBaseUrl,
	CfgNetworkShareBaseUrl,
	ConfigValue,
	DocRef,
} from 'generated/stoserver/stoservertypes_types';
import { AdminLayout } from 'layout/AdminLayout';
import * as React from 'react';

interface FuseServerPageState {
	baseUrl: Result<ConfigValue>;
	networkShareBaseUrl: Result<ConfigValue>;
}

export default class FuseServerPage extends React.Component<{}, FuseServerPageState> {
	state: FuseServerPageState = {
		baseUrl: new Result<ConfigValue>((_) => {
			this.setState({ baseUrl: _ });
		}),
		networkShareBaseUrl: new Result<ConfigValue>((_) => {
			this.setState({ networkShareBaseUrl: _ });
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
			<AdminLayout title="FUSE server &amp; network folders" breadcrumbs={[]}>
				<Panel
					heading={
						<div>
							Settings{' '}
							<DocLink doc={DocRef.DocsDataInterfacesNetworkFoldersIndexMd} />
						</div>
					}>
					{this.renderSettings()}
				</Panel>
			</AdminLayout>
		);
	}

	private renderSettings() {
		return (
			<div className="form-horizontal">
				{this.state.baseUrl.draw((baseUrl) => (
					<div className="form-group">
						<label className="col-sm-2 control-label">
							FUSE server base URL &nbsp;
							<CommandIcon command={ConfigSetFuseServerBaseurl(baseUrl.Value)} />
						</label>
						<div className="col-sm-10">
							{baseUrl.Value !== ''
								? baseUrl.Value
								: 'Not set - unable to mount network folders'}
						</div>
					</div>
				))}

				{this.state.networkShareBaseUrl.draw((networkShareBaseUrl) => (
					<div className="form-group">
						<label className="col-sm-2 control-label">
							Network share base URL &nbsp;
							<CommandIcon
								command={ConfigSetNetworkShareBaseUrl(networkShareBaseUrl.Value)}
							/>
						</label>
						<div className="col-sm-10">
							{networkShareBaseUrl.Value !== ''
								? networkShareBaseUrl.Value
								: 'Not set'}
						</div>
					</div>
				))}

				<CommandButton command={FuseUnmountAll()} />
			</div>
		);
	}

	private fetchData() {
		this.state.baseUrl.load(() => getConfig(CfgFuseServerBaseUrl));
		this.state.networkShareBaseUrl.load(() => getConfig(CfgNetworkShareBaseUrl));
	}
}
