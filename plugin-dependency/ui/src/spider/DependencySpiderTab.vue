<template>
  <div class="dependency-spider-tab">
    <div class="top-bar">
      <cl-form
          :model="spiderData"
          inline
      >
        <cl-form-item :label="t('spider.dependencyType')">
          <cl-tag
              :label="spiderDataDependencyTypeLabel"
              :type="spiderDataDependencyTypeType"
              :tooltip="spiderDataDependencyTypeTooltip"
              size="normal"
          />
        </cl-form-item>
      </cl-form>
      <cl-button
          class="action-btn"
          :tooltip="installButtonTooltip"
          :disabled="!spiderData.dependency_type"
          @click="onInstallByConfig"
      >
        <font-awesome-icon class="icon" :icon="['fa', 'download']"/>
        {{ t('actions.install') }}
      </cl-button>
    </div>
    <cl-table
        :data="tableData"
        :columns="tableColumns"
        hide-footer
    />
    <InstallForm
        :visible="dialogVisible.install"
        :nodes="allNodes"
        :names="installForm.names"
        @confirm="onInstall"
        @close="() => onDialogClose('install')"
    />
    <UninstallForm
        :visible="dialogVisible.uninstall"
        :nodes="uninstallForm.nodes"
        :names="uninstallForm.names"
        @confirm="onUninstall"
        @close="() => onDialogClose('uninstall')"
    />
  </div>
</template>

<script lang="ts">
import {computed, defineComponent, h, onMounted, ref} from 'vue';
import {ClNavLink, ClNodeType, ClTag, useRequest} from 'crawlab-ui';
import {useRoute} from 'vue-router';
import {useStore} from 'vuex';
import InstallForm from '../components/form/InstallForm.vue';
import UninstallForm from '../components/form/UninstallForm.vue';
import {ElMessage, ElMessageBox} from 'element-plus';

const pluginName = 'dependency';
const t = (path) => window['_tp'](pluginName, path);
const _t = window['_t'];

const {
  get,
  post,
} = useRequest();

const endpoint = '/plugin-proxy/dependency';

export default defineComponent({
  name: 'DependencySpiderTab',
  components: {
    InstallForm,
    UninstallForm,
  },
  setup(props, {emit}) {
    const route = useRoute();

    const store = useStore();

    const allNodes = computed(() => store.state.node.allList);

    onMounted(() => store.dispatch(`node/getAllList`));

    const isInstallable = (dep) => {
      const {result} = dep;
      if (!result) return true;
      if (result.upgradable || result.downgradable) return true;
      const node_ids = result.node_ids || [];
      return node_ids.length < allNodes.value.length;
    };

    const isUninstallable = (dep) => {
      const {result} = dep;
      if (!result) return true;
      const node_ids = result.node_ids || [];
      return node_ids.length > 0;
    };

    const tableColumns = computed(() => {
      return [
        {
          key: 'name',
          label: t('table.columns.name'),
          icon: ['fa', 'font'],
          width: '200',
          value: (row) => h(ClNavLink, {
            label: row.name,
            path: `https://pypi.org/project/${row.name}`,
            external: true,
          }),
        },
        {
          key: 'version',
          label: t('table.columns.requiredVersion'),
          icon: ['fa', 'tag'],
          width: '200',
        },
        // {
        //   key: 'latest_version',
        //   label: 'Latest Version',
        //   icon: ['fa', 'tag'],
        //   width: '200',
        // },
        {
          key: 'versions',
          label: t('table.columns.installedVersion'),
          icon: ['fa', 'tag'],
          width: '200',
          value: (row) => {
            const res = [];
            if (!row.result || !row.result.versions) return;
            const {result} = row;
            if (!result) return;
            const {versions} = result;
            res.push(h('span', {style: 'margin-right: 5px'}, versions.join(', ')));
            if (result.upgradable) {
              res.push(h(ClTag, {
                type: 'primary',
                effect: 'light',
                size: 'mini',
                tooltip: t('common.upgradable'),
                icon: ['fa', 'arrow-up'],
              }));
            } else if (result.downgradable) {
              res.push(h(ClTag, {
                type: 'warning',
                effect: 'light',
                size: 'mini',
                tooltip: t('common.downgradable'),
                icon: ['fa', 'arrow-down'],
              }));
            }
            return res;
          },
        },
        {
          key: 'node_ids',
          label: t('table.columns.installedNodes'),
          icon: ['fa', 'server'],
          width: '580',
          value: (row) => {
            const result = row.result || {};
            let {node_ids} = result;
            if (!node_ids) return;
            return allNodes.value
                .filter(n => node_ids.includes(n._id))
                .map(n => {
                  return h(ClNodeType, {
                    isMaster: n.is_master,
                    label: n.name,
                  });
                });
          },
        },
        {
          key: 'actions',
          label: _t('components.table.columns.actions'),
          fixed: 'right',
          width: '200',
          buttons: (row) => {
            let {result} = row;
            if (!result) result = {};
            let tooltip;
            if (result.upgradable) {
              tooltip = t('actions.installAndUpgrade');
            } else if (result.downgradable) {
              tooltip = t('actions.installAndDowngrade');
            } else if (isInstallable(row)) {
              tooltip = t('actions.install');
            } else {
              tooltip = '';
            }
            return [
              {
                type: 'primary',
                icon: ['fa', 'download'],
                tooltip,
                disabled: (row) => !isInstallable(row),
                onClick: async (row) => {
                  installForm.value.names = [row.name];
                  dialogVisible.value.install = true;
                },
              },
              {
                type: 'danger',
                icon: ['fa', 'trash-alt'],
                tooltip: t('actions.uninstall'),
                disabled: (row) => !isUninstallable(row),
                onClick: async (row) => {
                  uninstallForm.value.names = [row.name];
                  dialogVisible.value.uninstall = true;
                },
              },
            ];
          },
          disableTransfer: true,
        },
      ];
    });

    const spiderData = ref({
      dependency_type: '',
      dependencies: [],
    });

    const tableData = computed(() => {
      if (!spiderData.value.dependencies) return [];
      return spiderData.value.dependencies;
    });

    const getSpiderData = async () => {
      const id = route.params.id;
      if (!id) return;
      const res = await get(`${endpoint}/spiders/${id}`);
      const {data} = res;
      spiderData.value = data;
    };

    onMounted(getSpiderData);

    const spiderDataDependencyTypeLabel = computed(() => {
      switch (spiderData.value.dependency_type) {
        case 'requirements.txt':
          return 'Python Pip';
        case 'package.json':
          return 'NPM';
        default:
          return t('spider.noDependencyType');
      }
    });

    const spiderDataDependencyTypeType = computed(() => {
      switch (spiderData.value.dependency_type) {
        case 'requirements.txt':
          return 'primary';
        case 'package.json':
          return 'primary';
        default:
          return 'info';
      }
    });

    const spiderDataDependencyTypeTooltip = computed(() => {
      switch (spiderData.value.dependency_type) {
        case 'requirements.txt':
          return t('spider.tooltip.requirementsTxt');
        case 'package.json':
          return t('spider.tooltip.packageJson');
        default:
          return t('spider.tooltip.other');
      }
    });

    const installButtonTooltip = computed(() => {
      switch (spiderData.value.dependency_type) {
        case 'requirements.txt':
          return t('spider.installButton.tooltip.requirementsTxt');
        case 'package.json':
          return t('spider.installButton.tooltip.packageJson');
        default:
          return t('spider.installButton.tooltip.other');
      }
    });

    const installForm = ref({
      names: [],
    });

    const uninstallForm = ref({
      nodes: [],
      names: [],
    });

    const dialogVisible = ref({
      install: false,
      uninstall: false,
    });

    const resetForms = () => {
      installForm.value = {
        names: [],
      };
      uninstallForm.value = {
        nodes: [],
        names: [],
      };
    };

    const onDialogClose = (key) => {
      dialogVisible.value[key] = false;
      resetForms();
    };

    const onInstall = async ({mode, upgrade, nodeIds}) => {
      const id = route.params.id;
      if (!id) return;
      const data = {
        mode,
        upgrade,
        names: installForm.value.names,
      };
      if (data.mode === 'all') {
        data['node_id'] = nodeIds;
      }
      await post(`${endpoint}/spiders/${id}/install`, data);
      await ElMessage.success('Started to install dependencies');
      onDialogClose('install');
    };

    const onUninstall = async ({mode, nodeIds}) => {
      const id = route.params.id;
      if (!id) return;
      const data = {
        names: uninstallForm.value.names,
        mode,
      };
      if (data.mode === 'all') {
        data['node_id'] = nodeIds;
      }
      await post(`${endpoint}/spiders/${id}/uninstall`, data);
      await ElMessage.success('Started to uninstall dependencies');
      onDialogClose('uninstall');
    };

    const onSelect = (rows) => {
      installForm.value.names = rows.map(d => d.name);
    };

    const onInstallByConfig = async () => {
      await ElMessageBox.confirm('Are you sure to install?', 'Install');
      const id = route.params.id;
      if (!id) return;
      const mode = 'all';
      const data = {
        mode,
        use_config: true,
        spider_id: id,
      };
      if (data.mode === 'all') {
        data['node_id'] = allNodes.value.map(d => d._id);
      }
      await post(`${endpoint}/spiders/${id}/install`, data);
      await ElMessage.success('Started to install dependencies');
    };

    return {
      tableColumns,
      tableData,
      spiderData,
      spiderDataDependencyTypeLabel,
      spiderDataDependencyTypeType,
      spiderDataDependencyTypeTooltip,
      allNodes,
      onSelect,
      onDialogClose,
      onInstall,
      onUninstall,
      dialogVisible,
      installForm,
      uninstallForm,
      installButtonTooltip,
      onInstallByConfig,
      t,
    };
  },
});
</script>

<style scoped>
.dependency-spider-tab .top-bar {
  display: flex;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid #e6e6e6;
}

.dependency-spider-tab .top-bar >>> .el-form-item {
  margin-bottom: 0;
}

.dependency-spider-tab .top-bar >>> .action-btn {
  margin-left: 10px;
}

.dependency-spider-tab .top-bar >>> .icon {
  margin-right: 5px;
}

.dependency-spider-tab >>> .el-table {
  border-top: none;
  border-left: none;
  border-right: none;
}
</style>
